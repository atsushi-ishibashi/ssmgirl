package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/atsushi-ishibashi/ssmgirl/svc"
	"github.com/atsushi-ishibashi/ssmgirl/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/urfave/cli"
)

func NewShellCommand() cli.Command {
	return cli.Command{
		Name:  "shell",
		Usage: "Run shell script via ssm",
		Flags: []cli.Flag{
			cli.StringSliceFlag{
				Name:  "instance",
				Usage: "instance targeted by command, more than 1",
			},
			cli.StringFlag{
				Name:  "workdir",
				Usage: "directory where command will be executed",
			},
			cli.StringSliceFlag{
				Name:  "cmd",
				Usage: "command will be executed",
			},
			cli.StringFlag{
				Name:  "path",
				Usage: "path to yaml config file",
			},
			cli.BoolFlag{
				Name:  "dry-run",
				Usage: "dry-run. print instanceIDS, work directory and commands",
			},
		},
		Action: func(c *cli.Context) error {
			if err := util.ConfigAWS(c); err != nil {
				return err
			}
			sh, err := newShell(c)
			if err != nil {
				return err
			}
			if c.Bool("dry-run") {
				sh.dryPrint()
			} else {
				sh.dryPrint()
				err = sh.execute()
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
}

type shell struct {
	ssmCli      *svc.SsmClient
	cmds        []string
	workDir     string
	instanceIDs []string
}

func newShell(c *cli.Context) (*shell, error) {
	sh := &shell{}
	//path flag
	if c.String("path") != "" {
		if err := sh.parseYaml(c.String("path")); err != nil {
			return nil, err
		}
	}
	//instance flag
	if len(sh.instanceIDs) > 0 && len(c.StringSlice("instance")) > 0 {
		return nil, util.ErrorRed(fmt.Sprint("--path and --instance conflict"))
	}
	if len(sh.instanceIDs) == 0 && len(c.StringSlice("instance")) == 0 {
		return nil, util.ErrorRed(fmt.Sprint("--instance is required, more than 1"))
	}
	if len(c.StringSlice("instance")) > 0 {
		sh.instanceIDs = c.StringSlice("instance")
	}
	//workdir flag
	if sh.workDir != "" && c.String("workdir") != "" {
		return nil, util.ErrorRed(fmt.Sprint("--path and --workdir conflict"))
	}
	if sh.workDir == "" && c.String("workdir") == "" {
		return nil, util.ErrorRed(fmt.Sprint("--workdir is required"))
	}
	if c.String("workdir") != "" {
		sh.workDir = c.String("workdir")
	}
	//cmd flag
	if len(sh.cmds) > 0 && len(c.StringSlice("cmd")) > 0 {
		return nil, util.ErrorRed(fmt.Sprint("--path and --cmd conflict"))
	}
	if len(sh.cmds) == 0 && len(c.StringSlice("cmd")) == 0 {
		return nil, util.ErrorRed(fmt.Sprint("--cmd is required, more than 1"))
	}
	if len(c.StringSlice("cmd")) > 0 {
		sh.cmds = c.StringSlice("cmd")
	}

	awsregion := os.Getenv("AWS_DEFAULT_REGION")
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	sh.ssmCli = &svc.SsmClient{SSM: ssm.New(sess, aws.NewConfig().WithRegion(awsregion))}

	if err = sh.validateInstances(); err != nil {
		return nil, err
	}
	return sh, nil
}

func (sh *shell) execute() error {
	sco, err := sh.ssmCli.RunShellScript(sh.cmds, sh.workDir, sh.instanceIDs)
	if err != nil {
		return err
	}
	for _, v := range sco.Command.InstanceIds {
		fmt.Printf("dispatch command to instanceID: %s\n", *v)
	}
	sh.waitUntilCommandFinish(sco)
	return nil
}

func (sh *shell) dryPrint() {
	var buff bytes.Buffer
	_, _ = buff.WriteString(fmt.Sprint("instances:\n"))
	for _, v := range sh.instanceIDs {
		_, _ = buff.WriteString(fmt.Sprintf("\t%s\n", v))
	}
	_, _ = buff.WriteString(fmt.Sprint("working directory:\n"))
	_, _ = buff.WriteString(fmt.Sprintf("\t%s\n", sh.workDir))
	_, _ = buff.WriteString(fmt.Sprint("command:\n"))
	for _, v := range sh.cmds {
		_, _ = buff.WriteString(fmt.Sprintf("\t%s\n", v))
	}
	fmt.Println(buff.String())
}

func (sh *shell) validateInstances() error {
	unavaInstances := []string{}
	avaiInstances, err := sh.ssmCli.ListAvailableInstanceIds()
	if err != nil {
		return err
	}
	for _, ci := range sh.instanceIDs {
		available := false
		for _, ai := range avaiInstances {
			if ai == ci {
				available = true
			}
		}
		if !available {
			unavaInstances = append(unavaInstances, ci)
		}
	}
	if len(unavaInstances) != 0 {
		return util.ErrorRed(fmt.Sprintf("instance %s is unavailable for ssm", unavaInstances))
	}
	return nil
}

type commandStatus struct {
	instanceID string
	status     string
}

type commandStatusList []*commandStatus

func (sh *shell) waitUntilCommandFinish(sco *ssm.SendCommandOutput) {
	csl := commandStatusList{}
	watchingInstances := []string{}
	for _, v := range sco.Command.InstanceIds {
		cs := &commandStatus{
			instanceID: *v,
			status:     *sco.Command.StatusDetails,
		}
		csl = append(csl, cs)
		watchingInstances = append(watchingInstances, *v)
	}
	for len(watchingInstances) > 0 {
		time.Sleep(3 * time.Second)
		nextWatchInstances := []string{}
		for _, wi := range watchingInstances {
			st, err := sh.ssmCli.GetCommandStatus(*sco.Command.CommandId, wi)
			if err != nil {
				util.SprintRed(fmt.Sprint(err))
				continue
			}
			for _, cs := range csl {
				if cs.instanceID == wi && st != cs.status {
					if st == "Pending" || st == "In Progress" || st == "Success" {
						util.PrintlnGreen(fmt.Sprintf("instanceID: %s change status %s -> %s", cs.instanceID, cs.status, st))
					} else if st == "Delayed" {
						util.PrintlnYellow(fmt.Sprintf("instanceID: %s change status %s -> %s", cs.instanceID, cs.status, st))
					} else {
						util.PrintlnRed(fmt.Sprintf("instanceID: %s change status %s -> %s", cs.instanceID, cs.status, st))
					}
					cs.status = st
				}
			}
			switch st {
			case "Pending", "In Progress", "Delayed":
				nextWatchInstances = append(nextWatchInstances, wi)
			}
		}
		watchingInstances = nextWatchInstances
	}
}

type YamlConfig struct {
	Instances []string `yaml:"instances"`
	WorkDir   string   `yaml:"workdir"`
	Commands  []string `yaml:"commands"`
}

func (sh *shell) parseYaml(path string) error {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	var yc YamlConfig
	if err = yaml.Unmarshal(buf, &yc); err != nil {
		return err
	}
	sh.instanceIDs = yc.Instances
	sh.workDir = yc.WorkDir
	sh.cmds = yc.Commands
	return nil
}

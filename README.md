# ssmgirl
ssmgirl is CLI to run command through AWS System Manager

## Usage
### shell
```
$ ssmgirl shell --help   
NAME:
   ssmgirl shell - Run shell script via ssm

USAGE:
   ssmgirl shell [command options] [arguments...]

OPTIONS:
   --instance value  instance targeted by command, more than 1
   --workdir value   directory where command will be executed
   --cmd value       command will be executed
   --path value      path to yaml config file
   --dry-run         dry-run. print instanceIDS, work directory and commands

Examples:
  $ ssmgirl --awsconf default shell --instance i-123456789 --workdir /home/ec2-user --cmd 'touch 1234.txt' --dry-run
  $ ssmgirl --awsconf default shell --instance i-123456789 --workdir /home/ec2-user --cmd 'touch 1234.txt'
  $ ssmgirl --awsconf default shell --path ./example/config.yaml
  $ ssmgirl --awsconf default shell --path ./example/config.yaml --instance i-123456789 //Error message shows when there are duplications between yaml and options
```

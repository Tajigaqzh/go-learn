package go29_cli_logging

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
)

var errUserNotFound = errors.New("user not found")

// Run 解析参数，将普通结果和诊断信息分别写入 stdout 与 stderr，并返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "version":
		fmt.Fprintln(stdout, "go-learn-cli v1.0.0")
		return 0
	case "user":
		return runUser(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "未知子命令 %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runUser(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("user", flag.ContinueOnError)
	flags.SetOutput(stderr)
	name := flags.String("name", "访客", "显示名称")
	verbose := flags.Bool("verbose", false, "输出诊断信息")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprintln(stderr, "用法: user [选项] <id>")
		return 2
	}
	id, err := strconv.Atoi(flags.Arg(0))
	if err != nil || id <= 0 {
		fmt.Fprintf(stderr, "无效用户 ID %q：必须是正整数\n", flags.Arg(0))
		return 2
	}
	if id == 404 {
		fmt.Fprintf(stderr, "查询用户 %d 失败: %v\n", id, errUserNotFound)
		return 1
	}
	if *verbose {
		fmt.Fprintf(stderr, "正在查询 user_id=%d\n", id)
	}
	fmt.Fprintf(stdout, "用户: id=%d name=%s\n", id, *name)
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "用法: go-learn-cli <version|user> [选项]")
}

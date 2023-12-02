package cmd

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type CLI struct {
	root *cobra.Command
}

var envFileTmpl = template.Must(template.New("env_file").Parse(`B2_ACCOUNT_ID=op://{{.Vault}}/{{.Item}}/B2_ACCOUNT_ID
B2_ACCOUNT_KEY=op://{{.Vault}}/{{.Item}}/B2_ACCOUNT_KEY
RESTIC_PASSWORD=op://{{.Vault}}/{{.Item}}/password
RESTIC_REPOSITORY=op://{{.Vault}}/{{.Item}}/RESTIC_REPOSITORY
RESTIC_PROGRESS_FPS=0.16666`))

func Build() *CLI {
	root := &cobra.Command{
		Use:   "backup",
		Short: "",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vault := viper.GetString("vault")
			item := viper.GetString("item")

			filename := "/tmp/.env.restic"
			err := writeEnvFile(filename, vault, item)
			if err != nil {
				return err
			}

			resticCmd, resticArgs, err := resticBackup(
				include(args...),
				exclude(viper.GetStringSlice("exclude")...),
				tags(viper.GetStringSlice("tag")...),
				verbose(viper.GetBool("verbose")),
			)
			if err != nil {
				return err
			}

			opCmd, opArgs := opRun(filename, resticCmd, resticArgs...)

			ctx := cmd.Context()
			opRunResticCmd := exec.CommandContext(ctx, opCmd, opArgs...)
			opRunResticCmd.Stderr = cmd.ErrOrStderr()
			opRunResticCmd.Stdout = cmd.OutOrStdout()

			return opRunResticCmd.Run()
		},
	}

	root.Flags().String("vault", "", "1Password Vault")
	root.Flags().String("item", "", "1Password Item")
	root.Flags().StringSliceP("exclude", "e", nil, "Exclude globs")
	root.Flags().StringSliceP("tag", "t", nil, "Tag backup")
	root.Flags().BoolP("verbose", "v", false, "Verbose logging")

	root.MarkFlagRequired("vault")
	root.MarkFlagRequired("item")

	viper.BindPFlags(root.Flags())

	return &CLI{
		root: root,
	}
}

type OnError func(error)

func LogFatal(err error) {
	log.Fatal(err)
}

func CheckError(f OnError, err error) {
	if err == nil {
		return
	}
	f(err)
}

func (c *CLI) Run(args ...string) error {
	if len(args) == 0 {
		args = os.Args[1:]
	}
	c.root.SetArgs(args)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	return c.root.ExecuteContext(ctx)
}

func writeEnvFile(filename, vault, item string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return envFileTmpl.Execute(f, struct {
		Vault string
		Item  string
	}{
		Vault: vault,
		Item:  item,
	})
}

func opRun(filename string, cmd string, args ...string) (string, []string) {
	opArgs := []string{"run", "--env-file", filename, "--", cmd}
	opArgs = append(opArgs, args...)
	return "op", opArgs
}

type resticBackupOptions struct {
	includes []string
	excludes []string
	tags     []string
	verbose  bool
}

type resticBackupOption func(*resticBackupOptions)

func include(ss ...string) resticBackupOption {
	return func(rbo *resticBackupOptions) {
		rbo.includes = ss
	}
}

func exclude(ss ...string) resticBackupOption {
	return func(rbo *resticBackupOptions) {
		rbo.excludes = ss
	}
}

func tags(ss ...string) resticBackupOption {
	return func(rbo *resticBackupOptions) {
		rbo.tags = ss
	}
}

func verbose(b bool) resticBackupOption {
	return func(rbo *resticBackupOptions) {
		rbo.verbose = true
	}
}

func resticBackup(opts ...resticBackupOption) (string, []string, error) {
	rbo := &resticBackupOptions{}
	for _, opt := range opts {
		opt(rbo)
	}

	includeFileName := "/tmp/.backupkeep"
	err := writeLines(includeFileName, rbo.includes)
	if err != nil {
		return "", nil, err
	}

	excludeFileName := "/tmp/.backupignore"
	err = writeLines(excludeFileName, rbo.excludes)
	if err != nil {
		return "", nil, err
	}

	args := []string{
		"-o",
		"b2.connections=10",
		"backup",
		"--read-concurrency",
		"10",
		"--exclude-file",
		excludeFileName,
		"--files-from",
		includeFileName,
	}
	for _, tag := range rbo.tags {
		args = append(args, "--tag", tag)
	}
	if rbo.verbose {
		args = append(args, "-v")
	}
	return "restic", args, nil
}

func writeLines(filename string, ss []string) error {
	var buf bytes.Buffer
	for _, s := range ss {
		buf.WriteString(s)
		buf.WriteRune('\n')
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, &buf)
	return err
}

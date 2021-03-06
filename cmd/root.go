package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cobra"
	"tcw.im/ufc"
)

var (
	// a global config file of rtfd, default is ~/.rtfd.cfg
	cfgFile string
	// rtfd version when building
	version string
	// commitID is git commit hash when building
	commitID string
	// built is UTC time when building
	built string

	showVersion bool
	showVerbose bool
	newInit     bool
)

var rootCmd = &cobra.Command{
	Use:   "rtfd",
	Short: "Build, read your exclusive and fuck docs.",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if showVerbose {
			fmt.Printf("v%s commit/%s built/%s\n", version, commitID, built)
		} else if showVersion {
			fmt.Println(version)
		} else if newInit {
			//新增rtfd配置文件
			if !ufc.IsFile(cfgFile) {
				tpl, err := rtfdConfigTPL()
				if err != nil {
					fmt.Printf("unable to read configuration template")
					os.Exit(128)
				}
				err = ioutil.WriteFile(cfgFile, tpl, 0644)
				if err != nil {
					fmt.Println("failed to generate configuration file")
					os.Exit(129)
				}
			} else {
				fmt.Printf("The rtfd config file(%s) already exists\n", cfgFile)
			}
		} else {
			cmd.Help()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	cfg, err := homedir.Expand("~/.rtfd.cfg")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().StringVarP(
		&cfgFile, "config", "c", cfg, "rtfd配置文件",
	)
	rootCmd.Flags().BoolVarP(
		&showVersion, "version", "v", false, "显示版本",
	)
	rootCmd.Flags().BoolVarP(
		&showVerbose, "info", "i", false, "显示版本与构建信息",
	)
	rootCmd.Flags().BoolVarP(
		&newInit, "init", "", false, "初始化rtfd配置文件",
	)
}

func initConfig() {
	if showVersion || showVerbose || newInit {
		return
	}
	// 除 -h/help 和根命令 -v/--init 选项外，其他子命令均需配置文件存在
	if cfgFile == "" || !ufc.IsFile(cfgFile) {
		fmt.Printf(
			"No valid configuration file: %s\n"+
				"Please use `rtfd --init` to initialize it.\n", cfgFile,
		)
		os.Exit(127)
	}
}

func rtfdConfigTPL() (content []byte, err error) {
	statikFS, err := fs.New()
	if err != nil {
		return
	}

	r, err := statikFS.Open("/rtfd.cfg")
	if err != nil {
		return
	}
	defer r.Close()
	content, err = ioutil.ReadAll(r)
	if err != nil {
		return
	}

	return content, nil
}

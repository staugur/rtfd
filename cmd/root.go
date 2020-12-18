package cmd

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
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

	showVer bool
)

var rootCmd = &cobra.Command{
	Use:   "rtfd",
	Short: "Build, read your exclusive and fuck docs.",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if showVer {
			fmt.Printf("v%s commit/%s built/%s\n", version, commitID, built)
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

	rootCmd.PersistentFlags().StringVarP(
		&cfgFile, "config", "c", cfg, "rtfd配置文件",
	)
	rootCmd.Flags().BoolVarP(
		&showVer, "version", "v", false, "显示版本与构建信息",
	)
}

func initConfig() {
	if showVer {
		return
	}
	if cfgFile == "" {
		fmt.Println("Invalid config value")
		os.Exit(127)
	} else {
		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			fmt.Println("config file does not exist")
			os.Exit(128)
		}
	}
}

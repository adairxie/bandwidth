package cmd

import (
	"time"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "./bandwidth",
	Short: "calculate domain's bandwidth",
	Long:  `calculate domain's bandwidth`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "version information",
	Long:  `version information`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("v0.1.0")
	},
}

// Execute adds al child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happend once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().StringP("config", "c", "./datatransfer.json", "JSON format configuration file")
	viper.BindPFlag("configfile", rootCmd.PersistentFlags().Lookup("config"))
	viper.SetDefault("configfile", "./datatransfer.json")

	rootCmd.PersistentFlags().StringP("start", "s", "", "time index for fetching from elasticsearch")
	viper.BindPFlag("esorigintime", rootCmd.PersistentFlags().Lookup("start"))
	viper.SetDefault("esorigintime", time.Now())

	mergeViperServer()
}

func mergeViperServer() {
	//加载配置文件
	configfile := viper.Get("configfile").(string)
	viper.SetConfigFile(configfile)

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

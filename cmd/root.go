package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var (
	priv        string
	from        string
	to          string
	amount      uint64
	simulate    bool
	swap        string
	slippageBps uint16
	timeout     uint16
)

var maxSupportedTransactionVersion uint64 = 0

var rootCmd = &cobra.Command{
	Use:   "swap-tool",
	Short: "start swap on blockchain",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(jupCmd)

	jupCmd.Flags().StringVar(&priv, "priv", "", "private key")
	jupCmd.Flags().StringVar(&from, "from", "", "token in mint address")
	jupCmd.Flags().StringVar(&to, "to", "", "token out mint address")
	jupCmd.Flags().Uint64Var(&amount, "amount", 0, "amount")
	jupCmd.Flags().StringVar(&swap, "swap", "https://quote-api.jup.ag/v6", "swap contract address or entrypoint url")
	jupCmd.MarkFlagsRequiredTogether("priv", "from", "to", "amount")
	jupCmd.Flags().BoolVar(&simulate, "simulate", true, "simulate swap") // set this default to true for local test
	jupCmd.Flags().Uint16Var(&slippageBps, "slippageBps", 50, "slippage bps")
	jupCmd.Flags().Uint16Var(&timeout, "timeout", 30, "confirmation timeout in seconds")

	rootCmd.AddCommand(solanaCmd)
}

func initConfig() {

	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

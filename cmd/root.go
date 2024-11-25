package cmd

import (
	"fmt"
	"os"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var (
	simulate    bool
	chainRpc    string
	chainWss    string
	slippageBps uint16
	timeout     uint16
)

var (
	maxSupportedTransactionVersion uint64 = 0
	swapApi                        string = "https://quote-api.jup.ag/v6"
)

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

	jupCmd.Flags().StringVar(&chainRpc, "chainRpc", rpc.MainNetBeta_RPC, "solana rpc endpoint")
	jupCmd.Flags().StringVar(&chainWss, "chainWss", rpc.MainNetBeta_WS, "solana wss endpoint")
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

package cmd

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
)

var solanaCmd = &cobra.Command{
	Use:   "solana [flags] privateKey signature",
	Short: "test",
	Args:  cobra.MatchAll(cobra.ExactArgs(2), cobra.OnlyValidArgs),
	Run:   solanaRun,
}

func solanaRun(cmd *cobra.Command, args []string) {
	priv := args[0]
	sig := args[1]

	accountFrom, err := solana.PrivateKeyFromBase58(priv)
	if err != nil {
		panic(err)
	}
	signature, err := solana.SignatureFromBase58(sig)
	if err != nil {
		panic(err)
	}

	rpcClient := rpc.New(rpc.MainNetBeta_RPC)

	// Get the transaction
	opts := &rpc.GetTransactionOpts{
		MaxSupportedTransactionVersion: &maxSupportedTransactionVersion,
	}
	txRes, err := rpcClient.GetTransaction(context.TODO(), signature, opts)

	prettyStr := TransactionToString(accountFrom.PublicKey(), txRes)

	fmt.Println(prettyStr)
}

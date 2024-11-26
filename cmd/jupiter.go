package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/google/go-querystring/query"
	"github.com/kervinct/swap-tool/types"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

var jupCmd = &cobra.Command{
	Use:   "jupiter [flags] privateKey inputMint outputMint amountIn",
	Short: "swap on jupiter",
	Args:  cobra.MatchAll(cobra.ExactArgs(4), cobra.OnlyValidArgs),
	Run:   jupRun,
}

func jupRun(cmd *cobra.Command, args []string) {
	/// prepare for request
	priv := args[0]
	from := args[1]
	to := args[2]
	amount, err := strconv.ParseUint(args[3], 10, 64)
	if err != nil {
		log.Fatalf("Invalid amount: %v", err)
		os.Exit(1)
	}
	accountFrom, err := solana.PrivateKeyFromBase58(priv)
	if err != nil {
		log.Fatalf("Invalid private key format, should be base58 encoded string")
		os.Exit(1)
	}
	if !checkIsValidAddress(from, to) {
		os.Exit(1)
	}

	outputMint, err := solana.PublicKeyFromBase58(to)
	if err != nil {
		log.Fatalf("Invalid output mint address: %v", err)
		os.Exit(1)
	}
	dstTokenAccount, _, err := solana.FindAssociatedTokenAddress(accountFrom.PublicKey(), outputMint)
	if err != nil {
		log.Fatalf("Failed to find associated token account: %v", err)
		os.Exit(1)
	}

	//TODO
	// check token account, if not existed, then create it
	// from, _, err := solana.FindAssociatedTokenAddress(accountFrom.PublicKey(), inputMint)
	// to, _, err = solana.FindAssociatedTokenAddress(accountFrom.PublicKey(), outputMint)
	// builder := solana.NewTransactionBuilder()
	// builder.AddInstruction(token.NewInitializeAccountInstruction(from, inputMint, accountFrom.PublicKey(), SysVarRentPubkey))
	// builder.AddInstruction(token.NewInitializeAccountInstruction(to, outputMint, accountFrom.PublicKey(), SysVarRentPubkey))
	// builder.SetRecentBlockHash(hash)
	// tx, err := builder.Build()
	// tx.Sign(accountFrom)

	/// 1. initialize client
	rpcClient := rpc.New(chainRpc)

	wsClient, err := ws.Connect(context.Background(), chainWss)
	if err != nil {
		log.Fatalf("Failed to connect to Solana WS endpoint: %v", err)
		os.Exit(1)
	}

	/// 2. fetch swap transaction
	jupQuoteResponse, swappedTokenAmount, err := fetchTransaction(
		accountFrom.PublicKey().String(),
		dstTokenAccount.String(),
		from,
		to,
		amount,
	)
	if err != nil {
		log.Fatalf("Failed to fetch transaction: %v", err)
		os.Exit(1)
	}

	/// 3. build transaction
	swapTransaction, err := solana.TransactionFromBase64(jupQuoteResponse.Transaction)
	if err != nil {
		log.Fatalf("Failed to parse swap transaction: %v", err)
		os.Exit(1)
	}

	// This method is only available in solana-core v1.9 or newer, <= v1.8 should use GetRecentBLockhash
	recentBlockHash, err := rpcClient.GetLatestBlockhash(context.TODO(), rpc.CommitmentFinalized)
	if err != nil {
		log.Fatalf("Failed to get recent block hash: %v", err)
		os.Exit(1)
	}
	swapTransaction.Message.RecentBlockhash = recentBlockHash.Value.Blockhash
	sigs, err := swapTransaction.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if accountFrom.PublicKey().Equals(key) {
			return &accountFrom
		}
		return nil
	})
	if err != nil {
		log.Fatalln("Unable to sign transaction:", err)
		os.Exit(1)
	}

	if simulate {
		/// 4a. simulation

		simulateOpts := rpc.SimulateTransactionOpts{
			// SigVerify:  true,  // conflicts with ReplaceRecentBlockhash
			Commitment:             rpc.CommitmentFinalized,
			ReplaceRecentBlockhash: true,
		}
		res, err := rpcClient.SimulateTransactionWithOpts(context.TODO(), swapTransaction, &simulateOpts)
		if err != nil {
			log.Fatalf("Failed to send simulate transaction: %v", err)
			os.Exit(1)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Simulation mode:\n"))
		if res.Value.Err != nil {
			sb.WriteString(fmt.Sprintf("\tError: %s\n", res.Value.Err))
			for _, log := range res.Value.Logs {
				sb.WriteString(fmt.Sprintf("\t\t%s\n", log))
			}
		} else {
			sb.WriteString(fmt.Sprintf("simulation succeeded, swapped out: %d\n", swappedTokenAmount))
		}
		fmt.Println(sb.String())
	} else {
		/// 4b. send transaction
		fmt.Printf("Sending transaction and waiting for confirmed...\n\t"+
			"Confirmation will break after %d seconds\n", timeout)
		fmt.Printf("\tTransaction signature: %s\n", sigs[0].String())
		sOpts := rpc.TransactionOpts{
			SkipPreflight:       true,
			PreflightCommitment: rpc.CommitmentFinalized,
		}
		timeoutDur := time.Second * time.Duration(timeout)
		sig, err := confirm.SendAndConfirmTransactionWithOpts(
			context.TODO(),
			rpcClient,
			wsClient,
			swapTransaction,
			sOpts,
			&timeoutDur,
		)
		if err != nil {
			log.Fatalf("Failed to send and confirm the transaction\n\t"+
				"Attention:\n\t\tThe transaction may be confirmed or discarded, "+
				"you can confirm it by hand [click](https://explorer.solana.com/tx/%s)", sigs[0].String())
			os.Exit(1)
		}

		opts := rpc.GetTransactionOpts{
			MaxSupportedTransactionVersion: &maxSupportedTransactionVersion,
		}
		txRes, err := rpcClient.GetTransaction(context.TODO(), sig, &opts)
		if err != nil {
			log.Fatalf("Failed to get confirmed transaction: %v", err)
		}

		fmt.Println(TransactionToString(accountFrom.PublicKey(), txRes))
	}
}

func TransactionToString(user solana.PublicKey, txRes *rpc.GetTransactionResult) string {
	type TokenAmount struct {
		Address              solana.PublicKey
		BeforeUiAmountString string
		AfterUiAmountString  string
	}
	var (
		accountIndex   int
		preSolBalance  uint64
		postSolBalance uint64
		tokenBalances  map[uint16]TokenAmount = make(map[uint16]TokenAmount)
	)

	tx, err := txRes.Transaction.GetTransaction()
	if err != nil {
		return fmt.Sprintf("Failed to get transaction: %v", err)
	}
	for i, account := range tx.Message.AccountKeys {
		if account == user {
			accountIndex = i
			break
		}
		accountIndex = -1
	}

	if accountIndex < 0 {
		return "Failed to find user account in transaction"
	}

	preSolBalance = txRes.Meta.PreBalances[accountIndex]
	postSolBalance = txRes.Meta.PostBalances[accountIndex]

	for _, postToken := range txRes.Meta.PostTokenBalances {
		if *postToken.Owner == user {
			t, ok := tokenBalances[postToken.AccountIndex]
			if ok {
				t.AfterUiAmountString = postToken.UiTokenAmount.UiAmountString
				tokenBalances[postToken.AccountIndex] = t
			} else {
				tokenBalances[postToken.AccountIndex] = TokenAmount{
					Address:              tx.Message.AccountKeys[postToken.AccountIndex],
					AfterUiAmountString:  postToken.UiTokenAmount.UiAmountString,
					BeforeUiAmountString: "0",
				}
			}
		}
	}

	for _, preToken := range txRes.Meta.PreTokenBalances {
		if *preToken.Owner == user {
			t, ok := tokenBalances[preToken.AccountIndex]
			if ok {
				t.BeforeUiAmountString = preToken.UiTokenAmount.UiAmountString
				tokenBalances[preToken.AccountIndex] = t
			} else {
				tokenBalances[preToken.AccountIndex] = TokenAmount{
					Address:              tx.Message.AccountKeys[preToken.AccountIndex],
					BeforeUiAmountString: preToken.UiTokenAmount.UiAmountString,
					AfterUiAmountString:  "0",
				}
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(
		fmt.Sprintf("Transaction details:\n\t[Before SOL balance: %d lamports/%s SOL ===>  After SOL balance: %d lamports/%s SOL]\n",
			preSolBalance,

			decimal.NewFromUint64(preSolBalance).Div(decimal.NewFromInt(1e9)).String(),
			postSolBalance,
			decimal.NewFromUint64(postSolBalance).Div(decimal.NewFromInt(1e9)).String(),
		),
	)
	for _, tokenInfo := range tokenBalances {
		sb.WriteString(
			fmt.Sprintf(
				"\tToken address: %s\n\t\t[Before amount: %s ===> After amount: %s]\n",
				tokenInfo.Address,
				tokenInfo.BeforeUiAmountString,
				tokenInfo.AfterUiAmountString,
			),
		)
	}

	return sb.String()
}

func checkIsValidAddress(addresses ...string) bool {
	for _, addr := range addresses {
		if _, err := solana.PublicKeyFromBase58(addr); err != nil {
			log.Println("Invalid input address:", addr)
			return false
		}
	}
	return true
}

func fetchTransaction(user, dstTokenAccount, inputMint, outputMint string, amount uint64) (*types.JupSwapResponse, int64, error) {
	jupClient := &http.Client{}

	jupQuoteReq := types.NewJupQuoteRequest(inputMint, outputMint, slippageBps, amount)

	params, _ := query.Values(jupQuoteReq)
	// fmt.Printf("params: %s\n", params.Encode())
	quoteReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/quote?%s", swapApi, params.Encode()), nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
		return nil, 0, err
	}
	quoteReq.Header.Set("Accept", "application/json")

	quoteRes, err := jupClient.Do(quoteReq)
	if err != nil {
		log.Fatalf("Failed to send quote request: %v", err)
		return nil, 0, err

	}
	defer quoteRes.Body.Close()

	quoteBody, err := io.ReadAll(quoteRes.Body)
	if quoteRes.StatusCode != http.StatusOK {
		log.Fatalf("Failed to get quote, response message %s", string(quoteBody))
		return nil, 0, err
	}
	if err != nil {
		log.Fatalf("Failed to read quote response body: %v", err)
		return nil, 0, err

	}

	var jupQuoteRes types.JupQuoteResponse
	json.Unmarshal(quoteBody, &jupQuoteRes)

	jupSwapReq := types.NewJupSwapRequest(user, dstTokenAccount, jupQuoteRes)

	marshalled, err := json.Marshal(jupSwapReq)
	if err != nil {
		log.Fatalf("Failed to marshal swap request body: %v", err)
		return nil, 0, err
	}
	// fmt.Println("swap req marshalled: ", string(marshalled))
	swapReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/swap", swapApi), bytes.NewReader(marshalled))
	if err != nil {
		log.Fatalf("Failed to create swap request: %v", err)
		return nil, 0, err

	}
	swapReq.Header.Add("Accept", "application/json")
	swapReq.Header.Add("Content-Type", "application/json")

	swapRes, err := jupClient.Do(swapReq)
	if err != nil {
		log.Fatalf("Failed to send swap request: %v", err)
		return nil, 0, err
	}
	defer swapRes.Body.Close()

	swapBody, err := io.ReadAll(swapRes.Body)
	if swapRes.StatusCode != http.StatusOK {
		log.Fatalf("Failed to get swap, status: %d, response message %s", swapRes.StatusCode, string(swapBody))
		return nil, 0, err
	}
	if err != nil {
		log.Fatalf("Failed to read swap response body: %v", err)
		return nil, 0, err
	}
	// fmt.Println("swap body: ", string(swapBody))

	var jupSwapRes types.JupSwapResponse
	json.Unmarshal(swapBody, &jupSwapRes)

	outAmount, err := strconv.ParseInt(jupQuoteRes.OutAmount, 10, 64)
	if err != nil {
		log.Fatalf("Failed to parse out amount: %v", err)
		return nil, 0, err
	}

	return &jupSwapRes, outAmount, nil
}

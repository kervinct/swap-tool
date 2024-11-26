### Usage

```shell
swap-tool jupiter [--flags] <privKeyBase58> <inputMint> <outputMint> <amount>
```

tips: Simulation mode default enabled, which won't broadcast the transaction to blockchain network, you can pass `--simulation=false` to disable this mode

### Compilation

```shell
go build
```

### Development processes

1. After go over the jupiter's doc-site and github-repo, only found [api-doc](https://station.jup.ag/api-v6/get-quote) and some react code

2. Because lack of the source code of jupiter contract, trying to check if instructions exist in react code

3. Only found fetchSwapTransaction and useJupiterExchange method, which hiding calls to jupiter api behind a lot of logics

4. Now is clear, our development processes will be these steps below

   - build a swap transaction with Jupiter api

   - signs the transaction with input private key

     the blockhash in respond transaction may be too old or throw not found error, replace it with a newly latest blockhash

   - send and confirm transaction with Go SDK

     there is a design of `simulation` mode, which is configured through `--simulate=[bool]`(and default `true`), in this mode, we can check the transaction whether can pass the on-chain verification

     transaction confirmation was finished with two promises by promise.race in web3.js(a timeout promise and a checking-status promise), in Go we can use select-channel pattern

     in the web3.js, send_and_confirm was implemented by two consequent request, one is send_transaction, another one is signature_subscribe(with a timeout option)

   - parse the content of transaction

   the change of SOL balance is checked by go over the preBalances and postBalances field in the response

   and the change of Token balance if checked by go over the preTokenBalances and postTokenBalances field in the response

   ```shell
   # output example
   Sending transaction and waiting for confirmed...
   	Confirmation will break after 30 seconds
   	Transaction signature: 2aDwbyhA5z9roW2Pn1pi2BCHUUiSAkYLfi81kYYVGyAD6QJcvn9bbx8M2p57SoBRvGYBT3WeGcoaw7LMHdK8yipk
   Transaction details:
   	[Before SOL balance: 65814570 lamports/0.06581457 SOL ===>  After SOL balance: 55716234 lamports/0.055716234 SOL]
   	Token mint: Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB
   		account address: BkNorXevTL44k7DqPeiPkSwyaQwrtcBZtcjGdPmUEjaZ
   			[Before amount: 7.475433 ===> After amount: 9.844253]
   ```

5. Actual development
   - command-line was built by cobra
   - manage different swap platform with subcommands
   - Jupiter api was designed with two processes, first get the Quote, then use the Quote as part of swap request to get swap transaction, while the Quote.platformFee.amount always an empty string, which caused /swap api throws `{"error":"Failed to deserialize the JSON body into the target type"}`, force set to "0" solved

### Known issues(TBR)

1. We didn't check if there is a valid swap pool for input mint pairs
2. The swap transaction assumed token associated account is created, but we didn't handle that case yet, you can create the account with spl-token cli by hand(`spl-token create account <mint>`)
3. We didn't request the decimal of input mint(emit one more RPC call to network caused slightly latency), the input `amount` should be the value without decimal(.eg 0.01 SOL should be pass in 10000000)

### i18

[Chinese](https://github.com/kervinct/swap-tool/docs/README-chs.md)

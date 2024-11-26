### Usage

```shell
swap-tool jupiter [--flags] <privKeyBase58> <inputMint> <outputMint> <amount>
```

tips: 默认启用模拟交易模式，此模式下不会提交到链上，可通过 `--simulation=false` 关闭模拟

### Compilation

```shell
go build
```

### 开发过程

    首先找 jupiter 相关文档和代码仓库，找到[api 文档](https://station.jup.ag/api-v6/get-quote)

    未找到合约源码，考虑从其前端代码库入手检查是否有构造 instruction 相关代码

    frontend 代码中找到 fetchSwapTransaction，它其实也是去调用 api
        https://github.com/jup-ag/terminal
        https://www.npmjs.com/package/@jup-ag/react-hook?activeTab=code

    此时大致思路已经明了
        1. 通过 jup api 构造 transaction
        2. 通过 solana sdk 签名 transaction
            使用命令行输入的私钥完成签名
            配置 recent blockhash（api 响应中的 blockhash 经测试容易过期或 blockhash not found）
        3. 通过 solana json api 提交交易
            通过 simulation api 进行模拟测试
            通过 sendAndConfirmTransaction 完成交易提交和确认（提交后通过 wss 接口订阅通知）
                此步骤 sdk 原理是构造两个 promise 并通过 promise.race 择一完成
                go 中通过 select channel 模拟相似行为

    实际开发
    1. 命令行使用 cobra 构造
    2. 通过子命令方式管理不同 swap 平台
    3. 测试 jup api 过程中遇到 json deserialize 问题，后多次检查发现是 platformFee.amount 返回值为空字符串，直接使用会导致上述错误，在代码中强制修改为 "0" 后提交可获取到 swap transaction
    4. 测试提交交易时发现手续费值较低，代码成功率较低，成功提交的交易只有
        https://solscan.io/tx/34sEVtRHAezCo5AsnGTQBYKL5z7kZW4GWqbdQ9qJbUC4cBewgz9KNcTQYeQBzq8UJr7YMh4s65hVfp4wKEsaGgtP
    5. 调整 computeUnitPrice 为 auto 后交易可较快确认
        https://solscan.io/tx/2aDwbyhA5z9roW2Pn1pi2BCHUUiSAkYLfi81kYYVGyAD6QJcvn9bbx8M2p57SoBRvGYBT3WeGcoaw7LMHdK8yipk

已知问题（待解决）

1. 未对选择的 token-pair 检查链上是否存在相应的 swap pool
2. 用户 token associated account 不存在时需要额外创建，暂未添加相关代码，可通过 spl-token 命令行工具手工创建 `spl-token create account <mint>`
3. 未对选择的 inputMint 检查其 decimal，因此 amount 相当于不带 decimal 的数值（如 0.01 SOL 应输入 10000000）

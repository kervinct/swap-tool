package types

type JupQuoteRequest struct {
	InputMint   string `url:"inputMint"`
	OutputMint  string `url:"outputMint"`
	Amount      uint64 `url:"amount"`
	SlippageBps uint16 `url:"slippageBps,omitempty"`
	SwapMode    string `url:"swapMode,omitempty"`
	// non-exhaustive options, list it in future
}

func NewJupQuoteRequest(inputMint, outputMint string, slippageBps uint16, amount uint64) JupQuoteRequest {
	return JupQuoteRequest{
		InputMint:   inputMint,
		OutputMint:  outputMint,
		Amount:      amount,
		SlippageBps: slippageBps,
		SwapMode:    "ExactIn",
	}
}

type JupQuoteResponse struct {
	InputMint            string      `json:"inputMint"`
	InAmount             string      `json:"inAmount"`
	OutputMint           string      `json:"outputMint"`
	OutAmount            string      `json:"outAmount"`
	OtherAmountThreshold string      `json:"otherAmountThreshold"`
	SwapMode             string      `json:"swapMode"`
	SlippageBps          int32       `json:"slippageBps"`
	PlatformFee          PlatformFee `json:"platformFee"`
	PriceImpactPct       string      `json:"priceImpactPct"`
	RoutePlan            []RoutePlan `json:"routePlan"`
	ContextSlot          int         `json:"contextSlot"`
	TimeTaken            int         `json:"timeTaken"`
}

type JupSwapRequest struct {
	UserPublicKey             string           `json:"userPublicKey"`
	WrapAndUnwrapSol          bool             `json:"wrapAndUnwrapSol"`
	UseSharedAccounts         bool             `json:"useSharedAccounts"`
	FeeAccount                string           `json:"feeAccount,omitempty"`
	TrackingAccount           string           `json:"trackingAccount"`
	PrioritizationFeeLamports int              `json:"prioritizationFeeLamports"`
	AsLegacyTransaction       bool             `json:"asLegacyTransaction"`
	UseTokenLedger            bool             `json:"useTokenLedger"`
	DestinationTokenAccount   string           `json:"destinationTokenAccount"`
	DynamicComputeUnitLimit   bool             `json:"dynamicComputeUnitLimit"`
	SkipUserAccountsRpcCalls  bool             `json:"skipUserAccountsRpcCalls"`
	DynamicSlippage           DynamicSlippage  `json:"dynamicSlippage"`
	QuoteResponse             JupQuoteResponse `json:"quoteResponse"`
}

func NewJupSwapRequest(userPubkey, dstTokenAccount string, quoteResponse JupQuoteResponse) JupSwapRequest {
	// if we leave this auto(derive from quote response), then probably received 422 json parse error
	quoteResponse.PlatformFee.Amount = "0"
	return JupSwapRequest{
		UserPublicKey:             userPubkey,
		WrapAndUnwrapSol:          true,
		UseSharedAccounts:         true,
		TrackingAccount:           userPubkey,
		PrioritizationFeeLamports: 0,
		AsLegacyTransaction:       false,
		UseTokenLedger:            false,
		DestinationTokenAccount:   dstTokenAccount,
		DynamicComputeUnitLimit:   true,
		SkipUserAccountsRpcCalls:  true,
		DynamicSlippage:           DynamicSlippage{MinBps: 50, MaxBps: 300},
		QuoteResponse:             quoteResponse,
	}
}

type DynamicSlippage struct {
	MinBps int32 `json:"minBps"`
	MaxBps int32 `json:"maxBps"`
}

type PlatformFee struct {
	Amount string `json:"amount"`
	FeeBps int32  `json:"feeBps"`
}

type SwapInfo struct {
	AmmKey     string `json:"ammKey"`
	Label      string `json:"label"`
	InputMint  string `json:"inputMint"`
	OutputMint string `json:"outputMint"`
	InAmount   string `json:"inAmount"`
	OutAmount  string `json:"outAmount"`
	FeeAmount  string `json:"feeAmount"`
	FeeMint    string `json:"feeMint"`
}

type RoutePlan struct {
	SwapInfo SwapInfo `json:"swapInfo"`
	Percent  int32    `json:"percent"`
}

type JupSwapResponse struct {
	Transaction               string                `json:"swapTransaction"`
	LastValidBlockHeight      int                   `json:"lastValidBlockHeight"`
	PrioritizationFeeLamports uint64                `json:"prioritizationFeeLamports,omitempty"`
	DynamicSlippageReport     DynamicSlippageReport `json:"dynamicSlippageReport,omitempty"`
}

type DynamicSlippageReport struct {
	SlippageBps                  int32  `json:"slippageBps,omitempty"`
	OtherAmount                  int32  `json:"OtherAmount,omitempty"`
	SimulatedIncurredSlippageBps int32  `json:"simulatedIncurredSlippageBps,omitempty"`
	AmplificationRatio           string `json:"amplificationRatio,omitempty"`
}

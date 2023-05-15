package views

import (
	"github.com/CrocSwap/graphcache-go/models"
	"github.com/CrocSwap/graphcache-go/types"
)

type IViews interface {
	QueryUserTokens(chainId types.ChainId, user types.EthAddress) (UserTokensResponse, error)
}

type Views struct {
	Models *models.Models
}

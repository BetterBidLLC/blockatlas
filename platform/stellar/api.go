package stellar

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stellar/go/clients/horizon"
	"github.com/trustwallet/blockatlas/coin"
	"github.com/trustwallet/blockatlas/models"
	"github.com/trustwallet/blockatlas/platform/stellar/source"
	"github.com/trustwallet/blockatlas/util"
	"net/http"
	"strconv"
)

func Setup(router gin.IRouter) {
	router.Use(util.RequireConfig("stellar.api"))
	router.Use(func(c *gin.Context) {
		source.Client.URL = viper.GetString("stellar.api")
		c.Next()
	})
	router.GET("/:address", getTransactions)
}

func getTransactions(c *gin.Context) {
	s, err := source.GetTxsOfAddress(c.Param("address"))
	if apiError(c, err) {
		return
	}

	txs := make([]models.LegacyTx, 0)
	for _, srcTx := range s {
		legacy := models.LegacyTx{
			Id:          srcTx.Tx.Hash,
			BlockNumber: uint64(srcTx.Tx.Ledger),
			Timestamp:   strconv.FormatInt(srcTx.Tx.LedgerCloseTime.Unix(), 10),
			From:        srcTx.Tx.Account,
			To:          srcTx.Payment.Destination.Address(),
			Value:       strconv.FormatInt(int64(srcTx.Payment.Amount), 10),
			GasPrice:    strconv.FormatInt(int64(srcTx.Tx.FeePaid), 10),
			Coin:        coin.IndexXLM,
			Nonce:       0,
		}
		legacy.Init()
		txs = append(txs, legacy)
	}

	page := models.Response {
		Total: len(txs),
		Docs:  txs,
	}
	page.Sort()
	c.JSON(http.StatusOK, &page)
}

func apiError(c *gin.Context, err error) bool {
	if hErr, ok := err.(*horizon.Error); ok {
		if hErr.Problem.Type == "https://stellar.org/horizon-errors/bad_request" {
			c.String(http.StatusBadRequest, "Bad request!")
			return true
		} else {
			c.String(http.StatusBadRequest, hErr.Problem.Type)
			return true
		}
	}
	if err != nil {
		logrus.WithError(err).Warning("Stellar API request failed")
		c.String(http.StatusBadGateway, "Stellar API request failed")
		return true
	}
	return false
}
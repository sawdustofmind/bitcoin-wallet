package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sawdustofmind/bitcoin-wallet/backend/wallet"
)

func RegisterRoutes(r *gin.Engine, w *wallet.Wallet) {
	r.GET("/balance", func(c *gin.Context) {
		balance, err := w.GetBalance()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"balance": balance.ToBTC()})
	})

	r.GET("/address", func(c *gin.Context) {
		addr, err := w.GetNewAddress()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"address": addr})
	})

	r.GET("/utxos", func(c *gin.Context) {
		utxos, err := w.GetUTXOs()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"utxos": utxos})
	})
}

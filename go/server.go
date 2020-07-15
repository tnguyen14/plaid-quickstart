package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plaid/plaid-go/plaid"
)

func init() {
	if PLAID_PRODUCTS == "" {
		PLAID_PRODUCTS = "transactions"
	}
	if PLAID_COUNTRY_CODES == "" {
		PLAID_COUNTRY_CODES = "US"
	}
}

// Fill with your Plaid API keys - https://dashboard.plaid.com/account/keys
var (
	PLAID_CLIENT_ID     = os.Getenv("PLAID_CLIENT_ID")
	PLAID_SECRET        = os.Getenv("PLAID_SECRET")
	PLAID_PRODUCTS      = os.Getenv("PLAID_PRODUCTS")
	PLAID_COUNTRY_CODES = os.Getenv("PLAID_COUNTRY_CODES")
	// Parameters used for the OAuth redirect Link flow.
	//
	// Set PLAID_REDIRECT_URI to 'http://localhost:8000/oauth-response.html'
	// The OAuth redirect flow requires an endpoint on the developer's website
	// that the bank website should redirect to. You will need to configure
	// this redirect URI for your client ID through the Plaid developer dashboard
	// at https://dashboard.plaid.com/team/api.
	PLAID_REDIRECT_URI = os.Getenv("PLAID_REDIRECT_URI")

	// Use 'sandbox' to test with fake credentials in Plaid's Sandbox environment
	// Use `development` to test with real credentials while developing
	// Use `production` to go live with real users
	APP_PORT = os.Getenv("APP_PORT")
)

func createClient(environment plaid.Environment) (client *plaid.Client, err error) {
	return plaid.NewClient(plaid.ClientOptions{
		PLAID_CLIENT_ID,
		PLAID_SECRET,
		environment, // Available environments are Sandbox, Development, and Production
		&http.Client{},
	})
}

// We store the access_token in memory - in production, store it in a secure
// persistent data store.
var accessToken string
var itemID string

// The payment_token is only relevant for the UK Payment Initiation product.
// We store the payment_token in memory - in production, store it in a secure
// persistent data store.
var paymentToken string
var paymentID string

// For OAuth flows, the process looks as follows.
// 1. create a link token with the redirectURI (as white listed at https://dashboard.plaid.com/team/api).
// 2. Once the flow succeeds, redirectURI will be called with additional parameters (as dictated by OAuth standards and Plaid)
// 3. link is re-initialized with the link token (from step 1) and the additional parameters from step 2.
var lastLinkToken string

func getAccessToken(c *gin.Context) {
	publicToken := c.PostForm("public_token")
	client, err := createClient(plaid.Sandbox)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response, err := client.ExchangePublicToken(publicToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	accessToken = response.AccessToken
	itemID = response.ItemID

	fmt.Println("public token: " + publicToken)
	fmt.Println("access token: " + accessToken)
	fmt.Println("item ID: " + itemID)

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"item_id":      itemID,
	})
}

// This functionality is only relevant for the UK Payment Initiation product.
// Sets the payment token in memory on the server side. We generate a new
// payment token so that the developer is not required to supply one.
// This makes the quickstart easier to use.
func createLinkTokenWithPayment(c *gin.Context) {
	client, err := createClient(plaid.Sandbox)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	recipientCreateResp, err := client.CreatePaymentRecipient(
		"Harry Potter",
		"GB33BUKB20201555555555",
		&plaid.PaymentRecipientAddress{
			Street:     []string{"4 Privet Drive"},
			City:       "Little Whinging",
			PostalCode: "11111",
			Country:    "GB",
		})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	recipientID := recipientCreateResp.RecipientID

	paymentCreateResp, err := client.CreatePayment(recipientID, "payment-ref", plaid.PaymentAmount{
		Currency: "GBP",
		Value:    12.34,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	paymentID = paymentCreateResp.PaymentID

	paymentTokenCreateResp, err := client.CreatePaymentToken(paymentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	paymentToken = paymentTokenCreateResp.PaymentToken

	fmt.Println("payment token: " + paymentToken)
	fmt.Println("payment id: " + paymentID)

	linkToken, httpErr := fetchLinkToken(paymentID)
	if httpErr != nil {
		c.JSON(httpErr.errorCode, gin.H{"error": httpErr.Error()})
	}
	lastLinkToken = linkToken
	c.JSON(http.StatusOK, gin.H{
		"link_token": lastLinkToken,
	})
}

func auth(c *gin.Context) {
	client, err := createClient(plaid.Sandbox)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response, err := client.GetAuth(accessToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts": response.Accounts,
		"numbers":  response.Numbers,
	})
}

func accounts(c *gin.Context) {
	client, err := createClient(plaid.Sandbox)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response, err := client.GetAccounts(accessToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts": response.Accounts,
	})
}

func balance(c *gin.Context) {
	client, err := createClient(plaid.Sandbox)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response, err := client.GetBalances(accessToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts": response.Accounts,
	})
}

func item(c *gin.Context) {
	client, err := createClient(plaid.Sandbox)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response, err := client.GetItem(accessToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	institution, err := client.GetInstitutionByID(response.Item.InstitutionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item":        response.Item,
		"institution": institution.Institution,
	})
}

func identity(c *gin.Context) {
	client, err := createClient(plaid.Sandbox)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response, err := client.GetIdentity(accessToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"identity": response.Accounts,
	})
}

func transactions(c *gin.Context) {
	client, err := createClient(plaid.Sandbox)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// pull transactions for the past 30 days
	endDate := time.Now().Local().Format("2006-01-02")
	startDate := time.Now().Local().Add(-30 * 24 * time.Hour).Format("2006-01-02")

	response, err := client.GetTransactions(accessToken, startDate, endDate)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts":     response.Accounts,
		"transactions": response.Transactions,
	})
}

// This functionality is only relevant for the UK Payment Initiation product.
// Retrieve Payment for a specified Payment ID
func payment(c *gin.Context) {
	client, err := createClient(plaid.Sandbox)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response, err := client.GetPayment(paymentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment": response.Payment,
	})
}

func createPublicToken(c *gin.Context) {
	client, err := createClient(plaid.Sandbox)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Create a one-time use public_token for the Item.
	// This public_token can be used to initialize Link in update mode for a user
	publicToken, err := client.CreatePublicToken(accessToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"public_token": publicToken,
	})
}

var envMapping = map[string]plaid.Environment{
	"sandbox":     plaid.Sandbox,
	"production":  plaid.Production,
	"development": plaid.Development,
}

func createLinkToken(c *gin.Context) {
	linkToken, err := fetchLinkToken("")
	if err != nil {
		c.JSON(err.errorCode, gin.H{"error": err.error})
	}
	lastLinkToken = linkToken
	c.JSON(http.StatusOK, gin.H{"link_token": linkToken})
}

type httpError struct {
	errorCode int
	error     string
}

func (httpError *httpError) Error() string {
	return httpError.error
}

func fetchLinkToken(paymentID string) (string, *httpError) {
	env := "sandbox"
	countryCodes := strings.Split(PLAID_COUNTRY_CODES, ",")
	products := strings.Split(PLAID_PRODUCTS, ",")
	redirectURI := PLAID_REDIRECT_URI
	fmt.Println("args", map[string]interface{}{
		"env":          env,
		"countryCodes": countryCodes,
		"products":     products,
		"redirectURI":  redirectURI,
	})
	// TODO: oauthNonce
	mappedEnv, ok := envMapping[env]
	if !ok {
		return "", &httpError{errorCode: http.StatusBadRequest, error: "invalid environment. Valid environments are sandbox, production, an development"}
	}
	client, err := createClient(mappedEnv)
	if err != nil {
		return "", &httpError{
			errorCode: http.StatusInternalServerError,
			error:     err.Error(),
		}
	}
	configs := plaid.LinkTokenConfigs{
		User: &plaid.LinkTokenUser{
			ClientUserID: "user-id",
		},
		ClientName:   "Plaid Quickstart",
		Products:     products,
		CountryCodes: countryCodes,
		Language:     "en",
		RedirectUri:  redirectURI,
		// Webhook:               "https://example.com/webhook",
	}
	if len(paymentID) > 0 {
		configs.PaymentInitiation = &plaid.PaymentInitiation{
			PaymentID: paymentID,
		}
	}
	resp, err := client.CreateLinkToken(configs)
	if err != nil {
		return "", &httpError{
			errorCode: http.StatusBadRequest,
			error:     err.Error(),
		}
	}
	return resp.LinkToken, nil
}

func getLinkTokenForSession(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"link_token": lastLinkToken,
	})
}

func main() {
	if APP_PORT == "" {
		APP_PORT = "8000"
	}

	r := gin.Default()
	r.LoadHTMLFiles("templates/index.tmpl", "templates/oauth-response.tmpl")
	r.Static("/static", "./static")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"item_id":        itemID,
			"access_token":   accessToken,
			"plaid_products": PLAID_PRODUCTS,
		})
	})

	r.GET("/oauth-response.html", func(c *gin.Context) {
		c.HTML(http.StatusOK, "oauth-response.tmpl", gin.H{
			"plaid_environment": "sandbox", // Switch this environment
			"plaid_link_token":  lastLinkToken,
		})
	})

	r.POST("/set_access_token", getAccessToken)
	r.POST("/create_link_token_with_payment", createLinkTokenWithPayment)
	r.GET("/auth", auth)
	r.GET("/accounts", accounts)
	r.GET("/balance", balance)
	r.GET("/item", item)
	r.POST("/item", item)
	r.GET("/identity", identity)
	r.GET("/transactions", transactions)
	r.POST("/transactions", transactions)
	r.GET("/payment", payment)
	r.GET("/create_public_token", createPublicToken)
	r.POST("/create_link_token", createLinkToken)
	r.POST("/link_token_for_session", getLinkTokenForSession)

	r.Run(":" + APP_PORT)
}

package bkredis

// IClient -- defines interface for methods
type IClient interface {
	// Open(conn *RedisConnection)
	// Close()

	// SetToken(tokenType authpb.TokenType, identityHash string, appShopID int64, hashedToken, encryptedToken string) error
	// GetToken(tokenType authpb.TokenType, hashedToken string) (string, error)
	// RemoveToken(identityHash string, tokenType authpb.TokenType, hashedToken string) error
	// SetLoyaltyID(shopID, loyaltyID int64) error
	// GetLoyaltyID(shopID int64) ([]string, error)
	// RemoveLoyaltyID(shopID int64) error
	// SetAppShopLastShowPopUp(appshopID, lastSeen int64) error
	// GetAppShopLastShowPopup(appshopID int64) (int64, error)
	// SetAppShopDoItLater(appshopID, currentTime int64) error
	// GetAppShopDoItLater(appshopID int64) (int64, error)

	// GetUserTokenHash(identityHash string) (string, error)
	// Get(key string) (string, error)
	// HGet(key, field string) (string, error)
	// HSet(key, field string, value interface{}) error
	// Set(key string, value interface{}, expiration time.Duration) error
	// GetProduct(key string) (string, error)
	// Del(key string) error
	// HDel(key, field string) error

	// iCartTokenManager
	// iReviewManager
}

// // Client -- real implementation
// type Client struct {
// 	client redis.UniversalClient
// }

// // Open -- open connection to db
// func (rc *Client) Open(conn *RedisConnection) {
// 	rc.client = redis.NewClient(&redis.Options{
// 		Addr:     conn.address,
// 		Password: conn.password, // no password set
// 		DB:       conn.db,       // use default DB
// 	})

// 	pong, err := rc.client.Ping().Result()
// 	if err != nil {
// 		logger.BkLog.Fatal("Could not ping to redis, details: ", err)
// 	}
// 	logger.BkLog.Info("Ping to redis: ", pong)
// }

// // Close close the redis client
// func (rc *Client) Close() {
// 	if rc.client != nil {
// 		rc.client.Close()
// 	}
// }

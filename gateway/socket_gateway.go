package gateway

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zishang520/socket.io/v2/socket"
)

// CartService matches the interface from the services package without importing it to avoid circular dependency.
type CartService interface {
	GetCart(userID string) (map[string]interface{}, error)
	AddToCart(userID string, data map[string]interface{}) error
	RemoveFromCart(userID, itemID string) error
	UpdateItemQuantity(userID, itemID string, quantity int) error
	ClearCart(userID string) error
}

// CartServiceInstance is a global reference to the active CartService injected at startup.
var CartServiceInstance CartService

// SocketGateway coordinates real-time events for orders and cart updates using Socket.io.
type SocketGateway interface {
	GetServer() *socket.Server
	SendOrderStatusUpdate(userID string, order interface{})
	SendNewOrderNotification(order interface{})
	SendCartUpdate(userID string, cart interface{})
}

type socketGateway struct {
	server *socket.Server
}

// Instance acts as a globally accessible pointer to the initialized SocketGateway.
var Instance SocketGateway

// clientSessionStore stores session metadata (like userId and roles) linked to client socket IDs.
var clientSessionStore sync.Map

// InitSocketGateway boots the socket.io server and configures the namespaces.
func InitSocketGateway() SocketGateway {
	io := socket.NewServer(nil, nil)

	// Orders Namespace setup
	ordersNs := io.Of("/orders", nil)
	ordersNs.Use(func(client *socket.Socket, next func(*socket.ExtendedError)) {
		tokenStr := extractToken(client)
		if tokenStr == "" {
			next(socket.NewExtendedError("Authentication token is required", nil))
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			secret := os.Getenv("JWT_SECRET")
			if secret == "" {
				secret = "super-secret-key-12345"
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			next(socket.NewExtendedError("Invalid or expired token", nil))
			return
		}

		userID, _ := claims["sub"].(string)
		if userID == "" {
			userID, _ = claims["userId"].(string)
		}

		var roles []string
		if rolesInterface, exists := claims["roles"]; exists {
			if rolesSlice, ok := rolesInterface.([]interface{}); ok {
				for _, r := range rolesSlice {
					if rStr, ok := r.(string); ok {
						roles = append(roles, rStr)
					}
				}
			}
		}

		clientSessionStore.Store(client.Id(), map[string]interface{}{
			"userId": userID,
			"roles":  roles,
		})
		next(nil)
	})

	ordersNs.On("connection", func(args ...interface{}) {
		client := args[0].(*socket.Socket)
		userID := ""
		var roles []string

		if sessionVal, ok := clientSessionStore.Load(client.Id()); ok {
			if session, ok := sessionVal.(map[string]interface{}); ok {
				userID, _ = session["userId"].(string)
				roles, _ = session["roles"].([]string)
			}
		}

		client.Join(socket.Room(fmt.Sprintf("user_%s", userID)))

		isAdmin := false
		for _, r := range roles {
			if r == "ADMIN" {
				isAdmin = true
				break
			}
		}
		if isAdmin {
			client.Join("admins")
			log.Printf("[SocketGateway] Admin connected to orders: %s (User: %s)", client.Id(), userID)
		} else {
			log.Printf("[SocketGateway] Client connected to orders: %s (User: %s)", client.Id(), userID)
		}

		client.On("disconnect", func(reason ...interface{}) {
			clientSessionStore.Delete(client.Id())
			log.Printf("[SocketGateway] Client disconnected from orders: %s", client.Id())
		})
	})

	// Cart Namespace setup
	cartNs := io.Of("/cart", nil)
	cartNs.Use(func(client *socket.Socket, next func(*socket.ExtendedError)) {
		tokenStr := extractToken(client)
		if tokenStr == "" {
			next(socket.NewExtendedError("Authentication token is required", nil))
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			secret := os.Getenv("JWT_SECRET")
			if secret == "" {
				secret = "super-secret-key-12345"
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			next(socket.NewExtendedError("Invalid or expired token", nil))
			return
		}

		userID, _ := claims["sub"].(string)
		if userID == "" {
			userID, _ = claims["userId"].(string)
		}

		clientSessionStore.Store(client.Id(), map[string]interface{}{
			"userId": userID,
		})
		next(nil)
	})

	cartNs.On("connection", func(args ...interface{}) {
		client := args[0].(*socket.Socket)
		userID := ""
		if sessionVal, ok := clientSessionStore.Load(client.Id()); ok {
			if session, ok := sessionVal.(map[string]interface{}); ok {
				userID, _ = session["userId"].(string)
			}
		}

		client.Join(socket.Room(fmt.Sprintf("user_%s", userID)))
		log.Printf("[SocketGateway] Client connected to cart: %s (User: %s)", client.Id(), userID)

		// Send initial cart state
		if CartServiceInstance != nil {
			cart, err := CartServiceInstance.GetCart(userID)
			if err == nil {
				client.Emit("cartUpdated", cart)
			}
		}

		client.On("getCart", func(args ...interface{}) {
			log.Printf("[SocketGateway] [getCart] Received getCart event for user: %s", userID)
			if CartServiceInstance != nil {
				cart, err := CartServiceInstance.GetCart(userID)
				if err == nil {
					client.Emit("cartUpdated", cart)
				} else {
					log.Printf("[SocketGateway] [getCart] Error getting cart: %v", err)
				}
			}
		})

		client.On("addToCart", func(args ...interface{}) {
			log.Printf("[SocketGateway] [addToCart] Received addToCart event. Args: %v", args)
			if len(args) > 0 && CartServiceInstance != nil {
				var data map[string]interface{}
				if err := parseArgs(args[0], &data); err != nil {
					log.Printf("[SocketGateway] [addToCart] Error parsing args: %v", err)
					return
				}
				log.Printf("[SocketGateway] [addToCart] Parsed data: %+v", data)
				if err := CartServiceInstance.AddToCart(userID, data); err != nil {
					log.Printf("[SocketGateway] [addToCart] AddToCart error: %v", err)
				} else {
					log.Printf("[SocketGateway] [addToCart] AddToCart successful for user: %s", userID)
				}
			} else {
				log.Printf("[SocketGateway] [addToCart] Ignored. len(args)=%d, CartServiceInstance=%v", len(args), CartServiceInstance)
			}
		})

		client.On("removeItem", func(args ...interface{}) {
			log.Printf("[SocketGateway] [removeItem] Received removeItem event. Args: %v", args)
			if len(args) > 0 && CartServiceInstance != nil {
				if itemID, ok := args[0].(string); ok {
					if err := CartServiceInstance.RemoveFromCart(userID, itemID); err != nil {
						log.Printf("[SocketGateway] [removeItem] RemoveFromCart error: %v", err)
					} else {
						log.Printf("[SocketGateway] [removeItem] RemoveFromCart successful for item: %s", itemID)
					}
				} else {
					log.Printf("[SocketGateway] [removeItem] Error: arg is not a string: %v", args[0])
				}
			}
		})

		client.On("updateQuantity", func(args ...interface{}) {
			log.Printf("[SocketGateway] [updateQuantity] Received updateQuantity event. Args: %v", args)
			if len(args) > 0 && CartServiceInstance != nil {
				var data struct {
					ItemID   string `json:"itemId"`
					Quantity int    `json:"quantity"`
				}
				if err := parseArgs(args[0], &data); err == nil {
					if err := CartServiceInstance.UpdateItemQuantity(userID, data.ItemID, data.Quantity); err != nil {
						log.Printf("[SocketGateway] [updateQuantity] UpdateItemQuantity error: %v", err)
					} else {
						log.Printf("[SocketGateway] [updateQuantity] UpdateItemQuantity successful")
					}
				} else {
					log.Printf("[SocketGateway] [updateQuantity] Error parsing args: %v", err)
				}
			}
		})

		client.On("clearCart", func(args ...interface{}) {
			log.Printf("[SocketGateway] [clearCart] Received clearCart event")
			if CartServiceInstance != nil {
				if err := CartServiceInstance.ClearCart(userID); err != nil {
					log.Printf("[SocketGateway] [clearCart] ClearCart error: %v", err)
				} else {
					log.Printf("[SocketGateway] [clearCart] ClearCart successful")
				}
			}
		})

		client.On("disconnect", func(reason ...interface{}) {
			clientSessionStore.Delete(client.Id())
			log.Printf("[SocketGateway] Client disconnected from cart: %s. Reason: %v", client.Id(), reason)
		})
	})

	Instance = &socketGateway{server: io}
	return Instance
}

func (g *socketGateway) GetServer() *socket.Server {
	return g.server
}

func (g *socketGateway) SendOrderStatusUpdate(userID string, order interface{}) {
	ordersNs := g.server.Of("/orders", nil)
	ordersNs.To(socket.Room(fmt.Sprintf("user_%s", userID))).Emit("orderStatusUpdated", order)
	ordersNs.To("admins").Emit("orderStatusUpdated", order)
}

func (g *socketGateway) SendNewOrderNotification(order interface{}) {
	ordersNs := g.server.Of("/orders", nil)
	ordersNs.To("admins").Emit("newOrder", order)
}

func (g *socketGateway) SendCartUpdate(userID string, cart interface{}) {
	cartNs := g.server.Of("/cart", nil)
	cartNs.To(socket.Room(fmt.Sprintf("user_%s", userID))).Emit("cartUpdated", cart)
}

func parseArgs(arg interface{}, target interface{}) error {
	if str, ok := arg.(string); ok {
		return json.Unmarshal([]byte(str), target)
	}
	bytes, err := json.Marshal(arg)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, target)
}

func getHeader(headers map[string][]string, key string) string {
	if vals, ok := headers[key]; ok && len(vals) > 0 {
		return vals[0]
	}
	for k, vals := range headers {
		if strings.EqualFold(k, key) && len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

func extractToken(client *socket.Socket) string {
	handshake := client.Handshake()
	if handshake.Auth != nil {
		if authMap, ok := handshake.Auth.(map[string]interface{}); ok {
			if token, exists := authMap["token"].(string); exists {
				return token
			}
		}
		if authStr, ok := handshake.Auth.(string); ok {
			var authMap map[string]interface{}
			if err := json.Unmarshal([]byte(authStr), &authMap); err == nil {
				if token, exists := authMap["token"].(string); exists {
					return token
				}
			}
		}
	}

	if handshake.Headers != nil {
		if authHeader := getHeader(handshake.Headers, "Authorization"); authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				return parts[1]
			}
		}
	}
	return ""
}

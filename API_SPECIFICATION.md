# API Specification - refer_api

**Base URL**: `/api`

---

## 🔐 Authentication

### `POST /auth/register`
Creates a new user account.
- **Body**:
  ```json
  {
    "name": "Elias Rodriguez",
    "email": "elias@example.com",
    "password": "securepassword123",
    "keepUpdated": true
  }
  ```
- **Response** (201 Created):
  ```json
  {
    "id": "uuid-here",
    "name": "Elias Rodriguez",
    "email": "elias@example.com",
    "stars": 0,
    "photoUrl": null
  }
  ```

### `POST /auth/login`
Authenticates a user and returns a token.
- **Body**:
  ```json
  {
    "email": "admin@admin.com",
    "password": "admin"
  }
  ```
- **Response** (200 OK):
  ```json
  {
    "user": {
      "id": "uuid-here",
      "name": "Elias Rodriguez",
      "email": "admin@admin.com",
      "stars": 85
    },
    "accessToken": "jwt-token-string"
  }
  ```

### `POST /auth/forgot-password`
Sends a password reset link.
- **Body**:
  ```json
  {
    "email": "admin@admin.com"
  }
  ```
- **Response** (200 OK):
  ```json
  {
    "message": "If the email exists, a reset link will be sent."
  }
  ```

### `POST /auth/refresh`
Refreshes the access token using a refresh token.
- **Body**:
  ```json
  {
    "refreshToken": "string"
  }
  ```
- **Response** (200 OK):
  ```json
  {
    "accessToken": "new-access-token",
    "refreshToken": "old-or-new-refresh-token"
  }
  ```

### `GET /auth/validate`
Validates the current JWT token.
- **Header**: `Authorization: Bearer <token>`
- **Response** (200 OK):
  ```json
  {
    "valid": true,
    "user": { "userId": "uuid", "email": "admin@admin.com" }
  }
  ```

---

## 👤 User

### `GET /user/profile`
Gets the current user's profile details.
- **Header**: `Authorization: Bearer <token>`
- **Response** (200 OK):
  ```json
  {
    "id": "uuid",
    "name": "Elias Rodriguez",
    "email": "admin@admin.com",
    "stars": 85,
    "photoUrl": "https://...",
    "createdAt": "2024-04-10T..."
  }
  ```

---

## ☕ Products

### `GET /products`
Returns all available products.
- **Response** (200 OK):
  ```json
  [
    {
      "id": "uuid",
      "name": "Cascara Cold Brew",
      "price": 6.50,
      "category": "Café",
      "categoryIcon": "https://...",
      "rating": 4.9,
      "imageUrl": "...",
      "availableSizes": [
        { "id": "...", "name": "Pequeño", "price": 6.50 }
      ],
      "types": [
        { "id": "...", "name": "Whole Milk", "price": 0 }
      ]
    }
  ]
  ```

### `GET /products/seasonal`
Returns only the seasonal products from the special collection.
- **Response** (200 OK):
  ```json
  [
    {
      "id": "uuid",
      "name": "Spiced Autumn Latte",
      "description": "Warming seasonal notes",
      "price": 5.75,
      "imageUrl": "..."
    }
  ]
  ```

### `GET /products/top-rated`
Returns the top 5 highest-rated products.
- **Response** (200 OK):
  ```json
  [ { "id": "...", "name": "...", "rating": 4.9 } ]
  ```

### `GET /products/categories`
Returns a list of all product categories with their names and icon URLs.
- **Response** (200 OK):
  ```json
  [
    {
      "id": "uuid",
      "name": "Café",
      "iconUrl": "https://..."
    }
  ]
  ```

### `GET /products/:id`
Returns detailed information for a single product.
- **Response** (200 OK):
  ```json
  {
    "id": "uuid",
    "name": "Spiced Autumn Latte",
    "description": "...",
    "price": 5.75,
    "starsReward": 12,
    "category": "Café",
    "categoryIcon": "https://...",
    "availableSizes": [...],
    "types": [...]
  }
  ```

---

## ✨ Stars & Rewards

### `GET /stars/balance`
Gets current user's star count.
- **Header**: `Authorization: Bearer <token>`
- **Response** (200 OK):
  ```json
  { "stars": 85 }
  ```

### `GET /stars/history`
Returns history of star earnings and redemptions.
- **Header**: `Authorization: Bearer <token>`
- **Response** (200 OK):
  ```json
  [
    {
      "id": "uuid",
      "type": "redeem|earn",
      "title": "Butter Croissant",
      "stars": -50,
      "date": "2024-04-10T..."
    }
  ]
  ```

### `GET /stars/rewards`
List of rewards available for redemption.
- **Response** (200 OK):
  ```json
  [
    {
      "id": "uuid",
      "title": "Butter Croissant",
      "starsRequired": 50,
      "imageUrl": "..."
    }
  ]
  ```

### `POST /stars/redeem`
Redeems stars for a reward.
- **Header**: `Authorization: Bearer <token>`
- **Body**:
  ```json
  { "rewardId": "uuid-here" }
  ```
- **Response** (201 Created):
  ```json
  {
    "success": true,
    "message": "Redeemed Butter Croissant",
    "remainingStars": 35
  }
  ```

### `GET /perks`
Returns special perks and benefits.
- **Response** (200 OK):
  ```json
  [
    { "id": "birthday", "title": "Birthday Treat", "status": "active" }
  ]
  ```

---

## 📊 Dashboard

### `GET /dashboard/summary`
Unified summary for the home/dashboard view.
- **Header**: `Authorization: Bearer <token>`
- **Response** (200 OK):
  ```json
  {
    "user": { "name": "...", "photoUrl": "...", "stars": 85 },
    "stars": 85,
    "rewards": [...],
    "recentActivity": [...],
    "nextRewardProgress": 0.85
  }
  ```

---

## 🛒 Cart (Pure Socket)

The cart is handled entirely through WebSockets for real-time synchronization across devices.

### Namespace: `/cart`

**Connection**:
- **Authentication**: Pass JWT token in `auth: { token: 'jwt-token' }` or include in Authorization header.
- **Initial State**: Upon successful connection, the server immediately emits a `cartUpdated` event with the current cart contents.

**Rooms**:
- Users are automatically joined to the `user_<userId>` room.

**Events (Client -> Server)**:

#### `addToCart`
Adds a product to the cart.
- **Payload**:
  ```json
  {
    "productId": "uuid",
    "sizeId": "uuid",
    "typeId": "uuid",
    "enhancementIds": ["uuid1", "uuid2"],
    "quantity": 1
  }
  ```

#### `removeItem`
Removes a specific item from the cart.
- **Payload**: `"cart_item_uuid"` (The ID of the item in the cart, not the productId).

#### `clearCart`
Removes all items from the user's cart.
- **Payload**: None.

**Events (Server -> Client)**:

#### `cartUpdated`
Pushed whenever the cart changes (addition, removal, clear) or upon connection.
- **Payload**:
  ```json
  {
    "items": [
      {
        "id": "uuid",
        "name": "Cascara Cold Brew",
        "imageUrl": "...",
        "size": "Grande",
        "type": "Oat Milk",
        "enhancements": ["Extra Shot"],
        "quantity": 1,
        "unitPrice": 7.50,
        "totalPrice": 7.50
      }
    ],
    "total": 7.50
  }
  ```

---

## 🔍 Search

### `GET /search/referrals?q=coffee&category=All`
Search products and categories.
- **Query Params**: `q` (search term), `category` (filter)
- **Response** (200 OK):
  ```json
  [ { "id": "...", "name": "...", "category": "Coffee" } ]
  ```

### `GET /search/categories`
Returns a list of all unique product categories.
- **Response** (200 OK):
  ```json
  ["All", "Coffee", "Bakery", "Frappuccino"]
  ```

---

## ⚙️ System

### `GET /config`
App configuration, versions and feature flags.
- **Response** (200 OK):
  ```json
  {
    "version": "1.0.0",
    "features": { "starsEnabled": true, "referralsEnabled": true },
    "endpoints": { "termsUrl": "...", "privacyUrl": "..." }
  }
  ```

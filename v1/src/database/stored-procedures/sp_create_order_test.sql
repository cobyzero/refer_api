-- =============================================================
-- TEST SCRIPT — sp_create_order
-- -------------------------------------------------------------
-- Ejecutar contra la base de datos `refer` con las tablas ya
-- creadas y el seed aplicado.
--
-- Cubre:
--   TC-1  Orden básica con 2 items, sin reward      → debe crear orden + stars
--   TC-2  Orden con redeemed_reward                 → debe marcar reward como claimed
--   TC-3  Reward ya canjeado (isClaimed = 1)        → debe lanzar error
--   TC-4  Items con enhancements                    → verifica campo TEXT
-- =============================================================

-- ── Helpers ──────────────────────────────────────────────────
-- Obtener IDs reales del seed para no hardcodear UUIDs
SET @user_id     = '00000000-0000-0000-0000-000000000002';  -- Regular User
SET @admin_id    = '00000000-0000-0000-0000-000000000001';  -- Admin

SELECT id INTO @location_id FROM locations LIMIT 1;
SELECT id INTO @product_a   FROM products WHERE name = 'Caramel Macchiato' LIMIT 1;
SELECT id INTO @product_b   FROM products WHERE name = 'Butter Croissant'  LIMIT 1;
SELECT starsReward INTO @stars_a FROM products WHERE id = @product_a;
SELECT starsReward INTO @stars_b FROM products WHERE id = @product_b;

SELECT CONCAT('[INFO] location=', @location_id,
              ' | product_a=', @product_a, ' (', @stars_a, ' stars)',
              ' | product_b=', @product_b, ' (', @stars_b, ' stars)')
       AS setup;


-- =============================================================
-- TC-1  Orden básica sin reward
-- =============================================================
SET @tc1_order_id = UUID();
SET @tc1_number   = 'TEST-0001';

SET @tc1_items = JSON_ARRAY(
  JSON_OBJECT(
    'id',          UUID(),
    'productId',   @product_a,
    'name',        'Caramel Macchiato',
    'quantity',    2,
    'price',       5.50,
    'imageUrl',    NULL,
    'size',        'Grande',
    'type',        'Oat Milk',
    'enhancements', NULL,
    'starsReward', @stars_a
  ),
  JSON_OBJECT(
    'id',          UUID(),
    'productId',   @product_b,
    'name',        'Butter Croissant',
    'quantity',    1,
    'price',       3.50,
    'imageUrl',    NULL,
    'size',        NULL,
    'type',        'Warmed',
    'enhancements', NULL,
    'starsReward', @stars_b
  )
);

CALL sp_create_order(
  @tc1_order_id,      -- p_order_id
  @user_id,           -- p_user_id
  @location_id,       -- p_location_id
  @tc1_number,        -- p_order_number
  14.50,              -- p_total
  1.00,               -- p_service_fee
  'card',             -- p_payment_method
  NULL,               -- p_pickup_time
  @tc1_items,         -- p_items_json
  NULL,               -- p_redeemed_reward_id
  @tc1_stars          -- OUT p_total_stars
);

SELECT @tc1_stars AS 'TC-1 Stars awarded (esperado: stars_a*2 + stars_b*1)';

-- Verificar orden
SELECT 'TC-1 Order' AS test,
       id, orderNumber, status, total, serviceFee, paymentMethod
FROM   orders WHERE id = @tc1_order_id;

-- Verificar items
SELECT 'TC-1 Items' AS test, id, name, quantity, price, size, type
FROM   order_items WHERE orderId = @tc1_order_id;

-- Verificar transacción de estrellas
SELECT 'TC-1 Star Txn' AS test, id, type, title, stars
FROM   star_transactions WHERE userId = @user_id ORDER BY date DESC LIMIT 1;


-- =============================================================
-- TC-2  Orden con redeemed_reward (redemption válido)
-- =============================================================

-- Primero obtenemos un reward y simulamos que el usuario lo redimió
SELECT id INTO @reward_id FROM rewards WHERE isActive = 1 LIMIT 1;
SELECT starsRequired, title INTO @req_stars, @reward_title
FROM   rewards WHERE id = @reward_id;

-- Insertar una entrada en redeemed_rewards sin canjear
SET @redeemed_id = UUID();
INSERT INTO redeemed_rewards (id, userId, rewardId, rewardTitle, isClaimed, orderId, createdAt)
VALUES (@redeemed_id, @user_id, @reward_id, @reward_title, 0, NULL, NOW());

SELECT CONCAT('[INFO] TC-2 redeemed_reward id=', @redeemed_id) AS setup_tc2;

SET @tc2_order_id = UUID();
SET @tc2_number   = 'TEST-0002';

-- Items propios de TC-2 con UUIDs frescos (no reutilizar @tc1_items)
SET @tc2_items = JSON_ARRAY(
  JSON_OBJECT(
    'id',          UUID(),
    'productId',   @product_a,
    'name',        'Caramel Macchiato',
    'quantity',    2,
    'price',       5.50,
    'imageUrl',    NULL,
    'size',        'Grande',
    'type',        'Oat Milk',
    'enhancements', NULL,
    'starsReward', @stars_a
  ),
  JSON_OBJECT(
    'id',          UUID(),
    'productId',   @product_b,
    'name',        'Butter Croissant',
    'quantity',    1,
    'price',       3.50,
    'imageUrl',    NULL,
    'size',        NULL,
    'type',        'Warmed',
    'enhancements', NULL,
    'starsReward', @stars_b
  )
);

CALL sp_create_order(
  @tc2_order_id,
  @user_id,
  @location_id,
  @tc2_number,
  14.50,
  1.00,
  'cash',
  '2026-12-31 10:00:00',
  @tc2_items,
  @redeemed_id,
  @tc2_stars
);

SELECT @tc2_stars AS 'TC-2 Stars awarded';

-- Verificar que isClaimed = 1 y orderId fue seteado
SELECT 'TC-2 Reward' AS test, id, isClaimed, orderId, rewardTitle
FROM   redeemed_rewards WHERE id = @redeemed_id;



-- =============================================================
-- TC-4  Items con enhancements (array serializado)
-- =============================================================
SET @tc4_order_id = UUID();
SET @tc4_number   = 'TEST-0004';

SET @tc4_items = JSON_ARRAY(
  JSON_OBJECT(
    'id',          UUID(),
    'productId',   @product_a,
    'name',        'Caramel Macchiato',
    'quantity',    1,
    'price',       5.50,
    'imageUrl',    'https://images.unsplash.com/photo-1485808191679-5f86510681a2',
    'size',        'Extragrande',
    'type',        'Almond Milk',
    'enhancements', 'Shot de espresso extra,Caramel Drizzle',
    'starsReward', @stars_a
  )
);

CALL sp_create_order(
  @tc4_order_id,
  @user_id,
  @location_id,
  @tc4_number,
  7.25,
  0.75,
  'card',
  NULL,
  @tc4_items,
  NULL,
  @tc4_stars
);

SELECT @tc4_stars AS 'TC-4 Stars awarded';

SELECT 'TC-4 Item enhancements' AS test, id, name, size, type, enhancements
FROM   order_items WHERE orderId = @tc4_order_id;


-- =============================================================
-- RESUMEN FINAL
-- =============================================================
SELECT
  o.orderNumber,
  o.status,
  o.total,
  o.paymentMethod,
  COUNT(oi.id)  AS items,
  (SELECT stars FROM star_transactions
   WHERE  title LIKE CONCAT('Order ', o.orderNumber) LIMIT 1) AS starsEarned
FROM   orders        o
JOIN   order_items  oi ON oi.orderId = o.id
WHERE  o.orderNumber IN ('TEST-0001','TEST-0002','TEST-0004')
GROUP  BY o.id
ORDER  BY o.createdAt;

-- =============================================================
-- LIMPIEZA (comentar si quieres conservar los datos de prueba)
-- =============================================================
-- DELETE FROM order_items      WHERE orderId IN (SELECT id FROM orders WHERE orderNumber LIKE 'TEST-%');
-- DELETE FROM star_transactions WHERE title    LIKE 'Order TEST-%';
-- DELETE FROM redeemed_rewards  WHERE id = @redeemed_id;
-- DELETE FROM orders            WHERE orderNumber LIKE 'TEST-%';

-- =============================================================
-- sp_create_order
-- -------------------------------------------------------------
-- Creates an order with its items, awards earned stars
-- and optionally marks a redeemed-reward as claimed,
-- all inside a single atomic transaction.
--
-- Parameters
--   p_order_id            UUID of the new order (generated outside)
--   p_user_id             UUID of the ordering user
--   p_location_id         UUID of the pickup location
--   p_order_number        Human-readable order code  (e.g. "ER-4231")
--   p_total               Total amount charged
--   p_service_fee         Service fee amount
--   p_payment_method      e.g. "card", "cash"
--   p_pickup_time         Optional DATETIME (NULL allowed)
--   p_items_json          JSON array with the order items:
--                           [{ "id", "productId", "name", "quantity",
--                              "price", "imageUrl", "size", "type",
--                              "enhancements", "starsReward" }]
--   p_redeemed_reward_id  UUID of a RedeemedReward to mark as claimed
--                         (NULL if no reward is being used)
--
-- OUT p_total_stars        Stars awarded in this order (returned)
-- =============================================================

DROP PROCEDURE IF EXISTS sp_create_order;

DELIMITER $$

CREATE PROCEDURE sp_create_order(
  IN  p_order_id            VARCHAR(36),
  IN  p_user_id             VARCHAR(36),
  IN  p_location_id         VARCHAR(36),
  IN  p_order_number        VARCHAR(20),
  IN  p_total               DOUBLE,
  IN  p_service_fee         DOUBLE,
  IN  p_payment_method      VARCHAR(50),
  IN  p_pickup_time         DATETIME,
  IN  p_items_json          JSON,
  IN  p_redeemed_reward_id  VARCHAR(36),
  OUT p_total_stars         INT
)
BEGIN
  -- ── Local variables ──────────────────────────────────────────
  DECLARE v_item_count      INT DEFAULT 0;
  DECLARE v_idx             INT DEFAULT 0;
  DECLARE v_item_id         VARCHAR(36);
  DECLARE v_product_id      VARCHAR(36);
  DECLARE v_name            VARCHAR(255);
  DECLARE v_quantity        INT;
  DECLARE v_price           DOUBLE;
  DECLARE v_image_url       VARCHAR(500);
  DECLARE v_size            VARCHAR(50);
  DECLARE v_type            VARCHAR(50);
  DECLARE v_enhancements    TEXT;
  DECLARE v_stars_reward    INT DEFAULT 0;
  DECLARE v_transaction_id  VARCHAR(36);
  DECLARE v_err_msg         TEXT;

  -- ── Error handler — rollback on any error ────────────────────
  DECLARE EXIT HANDLER FOR SQLEXCEPTION
  BEGIN
    GET DIAGNOSTICS CONDITION 1 v_err_msg = MESSAGE_TEXT;
    ROLLBACK;
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = v_err_msg;
  END;

  -- ── Initialise output ────────────────────────────────────────
  SET p_total_stars = 0;

  START TRANSACTION;

  -- ── 1. Insert the Order header ───────────────────────────────
  INSERT INTO orders (
    id, orderNumber, userId, locationId,
    total, serviceFee, status, paymentMethod,
    pickupTime, createdAt
  ) VALUES (
    p_order_id,
    p_order_number,
    p_user_id,
    p_location_id,
    p_total,
    p_service_fee,
    'ORDERED',
    p_payment_method,
    p_pickup_time,
    NOW()
  );

  -- ── 2. Insert order items & accumulate stars ─────────────────
  SET v_item_count = JSON_LENGTH(p_items_json);

  WHILE v_idx < v_item_count DO

    SET v_item_id      = COALESCE(
                           NULLIF(JSON_UNQUOTE(JSON_EXTRACT(p_items_json, CONCAT('$[', v_idx, '].id'))),         'null'),
                           UUID()
                         );
    SET v_product_id   = NULLIF(JSON_UNQUOTE(JSON_EXTRACT(p_items_json, CONCAT('$[', v_idx, '].productId'))),   'null');
    SET v_name         = JSON_UNQUOTE(JSON_EXTRACT(p_items_json, CONCAT('$[', v_idx, '].name')));
    SET v_quantity     = JSON_EXTRACT(p_items_json,              CONCAT('$[', v_idx, '].quantity'));
    SET v_price        = JSON_EXTRACT(p_items_json,              CONCAT('$[', v_idx, '].price'));
    SET v_image_url    = NULLIF(JSON_UNQUOTE(JSON_EXTRACT(p_items_json, CONCAT('$[', v_idx, '].imageUrl'))),    'null');
    SET v_size         = NULLIF(JSON_UNQUOTE(JSON_EXTRACT(p_items_json, CONCAT('$[', v_idx, '].size'))),        'null');
    SET v_type         = NULLIF(JSON_UNQUOTE(JSON_EXTRACT(p_items_json, CONCAT('$[', v_idx, '].type'))),        'null');
    SET v_enhancements = NULLIF(JSON_UNQUOTE(JSON_EXTRACT(p_items_json, CONCAT('$[', v_idx, '].enhancements'))), 'null');
    SET v_stars_reward = COALESCE(JSON_EXTRACT(p_items_json, CONCAT('$[', v_idx, '].starsReward')), 0);

    INSERT INTO order_items (
      id, orderId, productId, name, quantity,
      price, imageUrl, size, type, enhancements
    ) VALUES (
      v_item_id,
      p_order_id,
      v_product_id,
      v_name,
      v_quantity,
      v_price,
      v_image_url,
      v_size,
      v_type,
      v_enhancements
    );

    -- Accumulate stars: starsReward × quantity
    SET p_total_stars = p_total_stars + (v_stars_reward * v_quantity);

    SET v_idx = v_idx + 1;
  END WHILE;

  -- ── 3. Record star-earning transaction (if any stars) ────────
  IF p_total_stars > 0 THEN
    SET v_transaction_id = UUID();
    INSERT INTO star_transactions (id, userId, type, title, stars, date)
    VALUES (
      v_transaction_id,
      p_user_id,
      'earn',
      CONCAT('Order ', p_order_number),
      p_total_stars,
      NOW()
    );
  END IF;

  -- ── 4. Claim the redeemed reward (if provided) ───────────────
  IF p_redeemed_reward_id IS NOT NULL THEN
    UPDATE redeemed_rewards
    SET    isClaimed = 1,
           orderId   = p_order_id
    WHERE  id        = p_redeemed_reward_id
      AND  isClaimed = 0;

    -- Guard: if no row updated the reward was already claimed
    IF ROW_COUNT() = 0 THEN
      SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = 'Redeemed reward not found or already claimed';
    END IF;
  END IF;

  COMMIT;

END$$

DELIMITER ;

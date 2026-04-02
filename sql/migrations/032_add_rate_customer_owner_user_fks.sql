-- 可选约束：为 rate_customer 四个归属字段增加 users 外键
-- 前置条件：先完成 030/031 审计与修复，确保不存在非法用户ID

ALTER TABLE rate_customer
  ADD CONSTRAINT fk_rate_customer_customer_fee_owner_user
    FOREIGN KEY (customer_fee_owner_id) REFERENCES users(id)
    ON DELETE SET NULL ON UPDATE CASCADE,
  ADD CONSTRAINT fk_rate_customer_network_line_fee_owner_user
    FOREIGN KEY (network_line_fee_owner_id) REFERENCES users(id)
    ON DELETE SET NULL ON UPDATE CASCADE,
  ADD CONSTRAINT fk_rate_customer_general_fee_owner_user
    FOREIGN KEY (general_fee_owner_id) REFERENCES users(id)
    ON DELETE SET NULL ON UPDATE CASCADE,
  ADD CONSTRAINT fk_rate_customer_channel_owner_user
    FOREIGN KEY (channel_owner_user_id) REFERENCES users(id)
    ON DELETE SET NULL ON UPDATE CASCADE;

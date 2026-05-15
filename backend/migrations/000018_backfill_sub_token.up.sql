-- Бэкфилл sub_token для существующих subscriptions.
-- Миграция 000016 добавила колонку как nullable; существующие строки
-- имели NULL, что ломает Go-сканирование в model.Subscription.SubToken
-- (тип string, не *string). Заодно вешаем NOT NULL чтобы впредь NULL'ов
-- не было.

UPDATE subscriptions
SET sub_token = gen_random_uuid()::text
WHERE sub_token IS NULL;

ALTER TABLE subscriptions
ALTER COLUMN sub_token SET NOT NULL;

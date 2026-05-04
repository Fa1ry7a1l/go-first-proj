DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'orders'
          AND column_name = 'accrual'
          AND data_type = 'double precision'
    ) THEN
        ALTER TABLE orders
            ALTER COLUMN accrual TYPE BIGINT
            USING ROUND(accrual * 100)::BIGINT;
    END IF;
END $$;

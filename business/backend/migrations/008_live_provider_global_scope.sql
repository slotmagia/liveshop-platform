-- Admin manages one Platform-wide Provider catalogue. Platform Admin module
-- sessions intentionally carry no shop App/Merchant context, so app_id=0 is
-- the reserved global partition. Positive app_id partitions remain valid for
-- future explicitly scoped contracts.

ALTER TABLE live_provider_catalogue
    DROP CHECK ck_live_provider_catalogue_app,
    ADD CONSTRAINT ck_live_provider_catalogue_app CHECK (app_id >= 0);

ALTER TABLE live_provider
    DROP CHECK ck_live_provider_app,
    ADD CONSTRAINT ck_live_provider_app CHECK (app_id >= 0);

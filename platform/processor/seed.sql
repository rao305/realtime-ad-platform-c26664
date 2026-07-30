-- Seed campaigns for local demos and end-to-end tests.
INSERT INTO campaigns (
    id, name, creative_id, bid_cpm, daily_budget, spent_today, active,
    interests, countries, freq_cap_hour, pacing_mul
) VALUES
    (
        'camp-gadgets', 'Gadget Launch', 'cre-gadgets-1', 2.50, 100.00, 0,
        TRUE, ARRAY['gadgets', 'ai'], ARRAY['US', 'CA'], 5, 1.0
    ),
    (
        'camp-sports', 'Sports Gear', 'cre-sports-1', 1.80, 50.00, 0,
        TRUE, ARRAY['sports'], ARRAY['US'], 3, 1.0
    ),
    (
        'camp-paused', 'Paused Brand', 'cre-paused-1', 5.00, 200.00, 0,
        FALSE, ARRAY['gadgets'], ARRAY['US'], 5, 1.0
    ),
    (
        'camp-exhausted', 'Exhausted Deal', 'cre-exhausted-1', 3.00, 10.00, 10.00,
        FALSE, ARRAY['ai'], ARRAY['US'], 5, 0.0
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO creatives (id, campaign_id, headline, body, click_url) VALUES
    ('cre-gadgets-1', 'camp-gadgets', 'Next-gen gadgets', 'Shop the launch', 'https://example.com/gadgets'),
    ('cre-sports-1', 'camp-sports', 'Game day gear', 'Free shipping today', 'https://example.com/sports'),
    ('cre-paused-1', 'camp-paused', 'Paused', 'Should not serve', 'https://example.com/paused'),
    ('cre-exhausted-1', 'camp-exhausted', 'Sold out', 'Should not serve', 'https://example.com/exhausted')
ON CONFLICT (id) DO NOTHING;

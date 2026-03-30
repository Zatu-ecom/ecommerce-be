-- Seed: 003_seed_geo_data.sql
-- Description: Core country and currency master data (ISO standards)
-- Environment: ALL (core data)
-- Created: 2025-12-28

-- ============================================================================
-- CURRENCIES (ISO 4217 Standard - Major Currencies)
-- ============================================================================
INSERT INTO currency (id, code, name, symbol, symbol_native, decimal_digits, is_active) VALUES
(1, 'USD', 'US Dollar', '$', '$', 2, TRUE),
(2, 'EUR', 'Euro', '€', '€', 2, TRUE),
(3, 'GBP', 'British Pound', '£', '£', 2, TRUE),
(4, 'INR', 'Indian Rupee', '₹', '₹', 2, TRUE),
(5, 'JPY', 'Japanese Yen', '¥', '￥', 0, TRUE),
(6, 'CNY', 'Chinese Yuan', '¥', 'CN¥', 2, TRUE),
(7, 'AUD', 'Australian Dollar', 'A$', '$', 2, TRUE),
(8, 'CAD', 'Canadian Dollar', 'CA$', '$', 2, TRUE),
(9, 'CHF', 'Swiss Franc', 'CHF', 'CHF', 2, TRUE),
(10, 'SGD', 'Singapore Dollar', 'S$', '$', 2, TRUE),
(11, 'AED', 'UAE Dirham', 'AED', 'د.إ', 2, TRUE),
(12, 'SAR', 'Saudi Riyal', 'SAR', 'ر.س', 2, TRUE),
(13, 'BRL', 'Brazilian Real', 'R$', 'R$', 2, TRUE),
(14, 'MXN', 'Mexican Peso', 'MX$', '$', 2, TRUE),
(15, 'ZAR', 'South African Rand', 'R', 'R', 2, TRUE),
(16, 'KRW', 'South Korean Won', '₩', '₩', 0, TRUE),
(17, 'RUB', 'Russian Ruble', '₽', '₽', 2, TRUE),
(18, 'SEK', 'Swedish Krona', 'kr', 'kr', 2, TRUE),
(19, 'NOK', 'Norwegian Krone', 'kr', 'kr', 2, TRUE),
(20, 'DKK', 'Danish Krone', 'kr', 'kr', 2, TRUE),
(21, 'NZD', 'New Zealand Dollar', 'NZ$', '$', 2, TRUE),
(22, 'HKD', 'Hong Kong Dollar', 'HK$', '$', 2, TRUE),
(23, 'MYR', 'Malaysian Ringgit', 'RM', 'RM', 2, TRUE),
(24, 'THB', 'Thai Baht', '฿', '฿', 2, TRUE),
(25, 'IDR', 'Indonesian Rupiah', 'Rp', 'Rp', 0, TRUE),
(26, 'PHP', 'Philippine Peso', '₱', '₱', 2, TRUE),
(27, 'PLN', 'Polish Zloty', 'zł', 'zł', 2, TRUE),
(28, 'TRY', 'Turkish Lira', '₺', '₺', 2, TRUE),
(29, 'ILS', 'Israeli Shekel', '₪', '₪', 2, TRUE),
(30, 'EGP', 'Egyptian Pound', 'E£', 'ج.م', 2, TRUE)
ON CONFLICT (code) DO NOTHING;

-- Reset sequence to avoid conflicts
SELECT setval('currency_id_seq', (SELECT MAX(id) FROM currency));

-- ============================================================================
-- COUNTRIES (ISO 3166-1 Standard - Major Countries)
-- ============================================================================
INSERT INTO country (id, code, code_alpha3, name, native_name, phone_code, region, flag_emoji, is_active) VALUES
-- North America
(1, 'US', 'USA', 'United States', 'United States', '+1', 'Americas', '🇺🇸', TRUE),
(2, 'CA', 'CAN', 'Canada', 'Canada', '+1', 'Americas', '🇨🇦', TRUE),
(3, 'MX', 'MEX', 'Mexico', 'México', '+52', 'Americas', '🇲🇽', TRUE),

-- Europe
(4, 'GB', 'GBR', 'United Kingdom', 'United Kingdom', '+44', 'Europe', '🇬🇧', TRUE),
(5, 'DE', 'DEU', 'Germany', 'Deutschland', '+49', 'Europe', '🇩🇪', TRUE),
(6, 'FR', 'FRA', 'France', 'France', '+33', 'Europe', '🇫🇷', TRUE),
(7, 'IT', 'ITA', 'Italy', 'Italia', '+39', 'Europe', '🇮🇹', TRUE),
(8, 'ES', 'ESP', 'Spain', 'España', '+34', 'Europe', '🇪🇸', TRUE),
(9, 'NL', 'NLD', 'Netherlands', 'Nederland', '+31', 'Europe', '🇳🇱', TRUE),
(10, 'BE', 'BEL', 'Belgium', 'België', '+32', 'Europe', '🇧🇪', TRUE),
(11, 'AT', 'AUT', 'Austria', 'Österreich', '+43', 'Europe', '🇦🇹', TRUE),
(12, 'CH', 'CHE', 'Switzerland', 'Schweiz', '+41', 'Europe', '🇨🇭', TRUE),
(13, 'SE', 'SWE', 'Sweden', 'Sverige', '+46', 'Europe', '🇸🇪', TRUE),
(14, 'NO', 'NOR', 'Norway', 'Norge', '+47', 'Europe', '🇳🇴', TRUE),
(15, 'DK', 'DNK', 'Denmark', 'Danmark', '+45', 'Europe', '🇩🇰', TRUE),
(16, 'FI', 'FIN', 'Finland', 'Suomi', '+358', 'Europe', '🇫🇮', TRUE),
(17, 'PL', 'POL', 'Poland', 'Polska', '+48', 'Europe', '🇵🇱', TRUE),
(18, 'IE', 'IRL', 'Ireland', 'Éire', '+353', 'Europe', '🇮🇪', TRUE),
(19, 'PT', 'PRT', 'Portugal', 'Portugal', '+351', 'Europe', '🇵🇹', TRUE),
(20, 'GR', 'GRC', 'Greece', 'Ελλάδα', '+30', 'Europe', '🇬🇷', TRUE),

-- Asia
(21, 'IN', 'IND', 'India', 'भारत', '+91', 'Asia', '🇮🇳', TRUE),
(22, 'CN', 'CHN', 'China', '中国', '+86', 'Asia', '🇨🇳', TRUE),
(23, 'JP', 'JPN', 'Japan', '日本', '+81', 'Asia', '🇯🇵', TRUE),
(24, 'KR', 'KOR', 'South Korea', '대한민국', '+82', 'Asia', '🇰🇷', TRUE),
(25, 'SG', 'SGP', 'Singapore', 'Singapore', '+65', 'Asia', '🇸🇬', TRUE),
(26, 'HK', 'HKG', 'Hong Kong', '香港', '+852', 'Asia', '🇭🇰', TRUE),
(27, 'TW', 'TWN', 'Taiwan', '台灣', '+886', 'Asia', '🇹🇼', TRUE),
(28, 'MY', 'MYS', 'Malaysia', 'Malaysia', '+60', 'Asia', '🇲🇾', TRUE),
(29, 'TH', 'THA', 'Thailand', 'ประเทศไทย', '+66', 'Asia', '🇹🇭', TRUE),
(30, 'ID', 'IDN', 'Indonesia', 'Indonesia', '+62', 'Asia', '🇮🇩', TRUE),
(31, 'PH', 'PHL', 'Philippines', 'Pilipinas', '+63', 'Asia', '🇵🇭', TRUE),
(32, 'VN', 'VNM', 'Vietnam', 'Việt Nam', '+84', 'Asia', '🇻🇳', TRUE),
(33, 'PK', 'PAK', 'Pakistan', 'پاکستان', '+92', 'Asia', '🇵🇰', TRUE),
(34, 'BD', 'BGD', 'Bangladesh', 'বাংলাদেশ', '+880', 'Asia', '🇧🇩', TRUE),
(35, 'LK', 'LKA', 'Sri Lanka', 'ශ්‍රී ලංකා', '+94', 'Asia', '🇱🇰', TRUE),
(36, 'NP', 'NPL', 'Nepal', 'नेपाल', '+977', 'Asia', '🇳🇵', TRUE),

-- Middle East
(37, 'AE', 'ARE', 'United Arab Emirates', 'الإمارات', '+971', 'Asia', '🇦🇪', TRUE),
(38, 'SA', 'SAU', 'Saudi Arabia', 'المملكة العربية السعودية', '+966', 'Asia', '🇸🇦', TRUE),
(39, 'IL', 'ISR', 'Israel', 'ישראל', '+972', 'Asia', '🇮🇱', TRUE),
(40, 'TR', 'TUR', 'Turkey', 'Türkiye', '+90', 'Asia', '🇹🇷', TRUE),
(41, 'QA', 'QAT', 'Qatar', 'قطر', '+974', 'Asia', '🇶🇦', TRUE),
(42, 'KW', 'KWT', 'Kuwait', 'الكويت', '+965', 'Asia', '🇰🇼', TRUE),
(43, 'BH', 'BHR', 'Bahrain', 'البحرين', '+973', 'Asia', '🇧🇭', TRUE),
(44, 'OM', 'OMN', 'Oman', 'عُمان', '+968', 'Asia', '🇴🇲', TRUE),

-- Oceania
(45, 'AU', 'AUS', 'Australia', 'Australia', '+61', 'Oceania', '🇦🇺', TRUE),
(46, 'NZ', 'NZL', 'New Zealand', 'New Zealand', '+64', 'Oceania', '🇳🇿', TRUE),

-- South America
(47, 'BR', 'BRA', 'Brazil', 'Brasil', '+55', 'Americas', '🇧🇷', TRUE),
(48, 'AR', 'ARG', 'Argentina', 'Argentina', '+54', 'Americas', '🇦🇷', TRUE),
(49, 'CL', 'CHL', 'Chile', 'Chile', '+56', 'Americas', '🇨🇱', TRUE),
(50, 'CO', 'COL', 'Colombia', 'Colombia', '+57', 'Americas', '🇨🇴', TRUE),

-- Africa
(51, 'ZA', 'ZAF', 'South Africa', 'South Africa', '+27', 'Africa', '🇿🇦', TRUE),
(52, 'EG', 'EGY', 'Egypt', 'مصر', '+20', 'Africa', '🇪🇬', TRUE),
(53, 'NG', 'NGA', 'Nigeria', 'Nigeria', '+234', 'Africa', '🇳🇬', TRUE),
(54, 'KE', 'KEN', 'Kenya', 'Kenya', '+254', 'Africa', '🇰🇪', TRUE),

-- Russia
(55, 'RU', 'RUS', 'Russia', 'Россия', '+7', 'Europe', '🇷🇺', TRUE)

ON CONFLICT (code) DO NOTHING;

-- Reset sequence to avoid conflicts
SELECT setval('country_id_seq', (SELECT MAX(id) FROM country));

-- ============================================================================
-- COUNTRY-CURRENCY MAPPINGS
-- ============================================================================
INSERT INTO country_currency (country_id, currency_id, is_primary) VALUES
-- Americas
(1, 1, TRUE),   -- US -> USD
(2, 8, TRUE),   -- Canada -> CAD
(3, 14, TRUE),  -- Mexico -> MXN
(47, 13, TRUE), -- Brazil -> BRL

-- Europe (EUR zone)
(5, 2, TRUE),   -- Germany -> EUR
(6, 2, TRUE),   -- France -> EUR
(7, 2, TRUE),   -- Italy -> EUR
(8, 2, TRUE),   -- Spain -> EUR
(9, 2, TRUE),   -- Netherlands -> EUR
(10, 2, TRUE),  -- Belgium -> EUR
(11, 2, TRUE),  -- Austria -> EUR
(16, 2, TRUE),  -- Finland -> EUR
(18, 2, TRUE),  -- Ireland -> EUR
(19, 2, TRUE),  -- Portugal -> EUR
(20, 2, TRUE),  -- Greece -> EUR

-- Europe (Non-EUR)
(4, 3, TRUE),   -- UK -> GBP
(12, 9, TRUE),  -- Switzerland -> CHF
(13, 18, TRUE), -- Sweden -> SEK
(14, 19, TRUE), -- Norway -> NOK
(15, 20, TRUE), -- Denmark -> DKK
(17, 27, TRUE), -- Poland -> PLN
(55, 17, TRUE), -- Russia -> RUB

-- Asia
(21, 4, TRUE),  -- India -> INR
(22, 6, TRUE),  -- China -> CNY
(23, 5, TRUE),  -- Japan -> JPY
(24, 16, TRUE), -- South Korea -> KRW
(25, 10, TRUE), -- Singapore -> SGD
(26, 22, TRUE), -- Hong Kong -> HKD
(28, 23, TRUE), -- Malaysia -> MYR
(29, 24, TRUE), -- Thailand -> THB
(30, 25, TRUE), -- Indonesia -> IDR
(31, 26, TRUE), -- Philippines -> PHP

-- Middle East
(37, 11, TRUE), -- UAE -> AED
(38, 12, TRUE), -- Saudi Arabia -> SAR
(39, 29, TRUE), -- Israel -> ILS
(40, 28, TRUE), -- Turkey -> TRY

-- Oceania
(45, 7, TRUE),  -- Australia -> AUD
(46, 21, TRUE), -- New Zealand -> NZD

-- Africa
(51, 15, TRUE), -- South Africa -> ZAR
(52, 30, TRUE)  -- Egypt -> EGP

ON CONFLICT DO NOTHING;

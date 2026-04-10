-- Sample data for database inventium (pos table).
-- Run after migrations. TRUNCATE allows repeated `make loaddata` in dev.

TRUNCATE pos RESTART IDENTITY;

INSERT INTO pos (name, location, description, total_sale_unit) VALUES
  (
    'Downtown Flagship',
    '123 Market St, San Francisco, CA',
    'Full-service retail counter with cafe and express checkout lanes.',
    18420
  ),
  (
    'Airport Concourse B',
    'SFO Terminal 2, Gate B12',
    'Travel retail kiosk; grab-and-go and duty-free eligible items.',
    9033
  ),
  (
    'Warehouse Outlet',
    '880 Industrial Pkwy, Oakland, CA',
    'High-volume outlet; bulk SKUs and seasonal clearance.',
    45200
  ),
  (
    'University Bookstore',
    'Campus Center, Berkeley, CA',
    'Textbooks, supplies, and branded merchandise for students and faculty.',
    6211
  ),
  (
    'Neighborhood Express',
    '45 Oak Ave, Palo Alto, CA',
    'Compact neighborhood store with self-checkout and same-day pickup.',
    12890
  );

CREATE TABLE "pos" (
  "id" bigserial PRIMARY KEY,
  "name" varchar NOT NULL,
  "location" varchar NOT NULL,
  "description" varchar NOT NULL,
  "total_sale_unit" bigint NOT NULL
);

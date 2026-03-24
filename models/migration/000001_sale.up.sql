CREATE TABLE "pos" (
  "id" bigserial PRIMARY KEY,
  "name" varchar(255) NOT NULL,
  "location" varchar(255) NOT NULL,
  "description" text NOT NULL,
  "total_sale_unit" bigint NOT NULL
);

-- RLS for Refer App on Supabase-managed PostgreSQL.
--
-- Arquitectura: Flutter -> Go -> PostgreSQL. Flutter NUNCA accede directo.
-- Por eso este RLS es deliberadamente restrictivo:
--   * RLS ENABLED en todas las tablas de negocio (public.*).
--   * CERO policies permisivas para anon/authenticated -> PostgREST directo
--     queda bloqueado por defecto (deny-all).
--   * Go se conecta como postgres (superuser, bypassa RLS) y toda la
--     autorizacion vive en el middleware Go (JWT + RequireRole ADMIN).
--
-- Aplicar (solo DESPUES de migrar datos y validar):
--   psql "$DATABASE_URL" -f supabase/rls.sql
--
-- Revertir (si algo falla):
--   Para cada tabla: ALTER TABLE <t> DISABLE ROW LEVEL SECURITY;

DO $$
DECLARE
  t text;
  tables text[] := ARRAY[
    'users', 'roles', 'users_roles',
    'locations',
    'product_categories', 'product_sizes', 'product_available_sizes',
    'product_types', 'product_enhancements',
    'products', 'seasonal_products', 'collections',
    'rewards', 'star_transactions', 'redeemed_rewards',
    'cart_items', 'cart_item_enhancements',
    'orders', 'order_items',
    'app_settings',
    'audit_logs'
  ];
BEGIN
  FOREACH t IN ARRAY tables LOOP
    -- Solo si la tabla existe (idempotente, re-ejecutable).
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema = 'public' AND table_name = t) THEN
      EXECUTE format('ALTER TABLE public.%I ENABLE ROW LEVEL SECURITY', t);
      RAISE NOTICE 'RLS enabled on public.%', t;
    ELSE
      RAISE NOTICE 'skip missing table public.%', t;
    END IF;
  END LOOP;
END
$$;

-- Intencionadamente SIN policies: con RLS habilitado y sin policies, los roles
-- anon/authenticated (PostgREST) no pueden leer ni escribir nada.
-- El service_role de Supabase tambien queda bloqueado (bypassrls solo lo tiene
-- postgres/superuser), lo cual es lo correcto: ni siquiera con SERVICE_ROLE_KEY
-- se accede a datos de negocio fuera de Go.
--
-- Si en el futuro un caso justificado requiere acceso directo (ej. realtime de
-- lectura), agregar policies MINIMAS por tabla aqui y auditarlas, nunca
-- "FOR ALL USING (true)".

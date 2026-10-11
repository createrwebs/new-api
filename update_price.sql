UPDATE options
SET value = (value::jsonb || '{"sd-turbo": 0.005, "sd-image": 0.005, "dall-e-2": 0.005}'::jsonb)::text
WHERE key = 'ModelPrice';

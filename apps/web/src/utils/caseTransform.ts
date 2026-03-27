/**
 * Recursively converts snake_case object keys to camelCase.
 * Used at the API boundary to transform backend responses (snake_case)
 * into frontend convention (camelCase).
 */
export function snakeToCamel<T>(obj: unknown): T {
  if (obj === null || obj === undefined) return obj as T;
  if (Array.isArray(obj)) return obj.map((item) => snakeToCamel(item)) as T;
  if (typeof obj !== 'object') return obj as T;

  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(obj as Record<string, unknown>)) {
    const camelKey = key.replace(/_([a-z])/g, (_, c: string) => c.toUpperCase());
    result[camelKey] = snakeToCamel(value);
  }
  return result as T;
}

/**
 * Recursively converts camelCase object keys to snake_case.
 * Used at the API boundary to transform frontend request params (camelCase)
 * into backend convention (snake_case).
 */
export function camelToSnake<T>(obj: unknown): T {
  if (obj === null || obj === undefined) return obj as T;
  if (Array.isArray(obj)) return obj.map((item) => camelToSnake(item)) as T;
  if (typeof obj !== 'object') return obj as T;

  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(obj as Record<string, unknown>)) {
    const snakeKey = key.replace(/[A-Z]/g, (c) => `_${c.toLowerCase()}`);
    result[snakeKey] = camelToSnake(value);
  }
  return result as T;
}

/**
 * Valid test image buffers for E2E tests
 *
 * These are actual valid image files that Go's image decoder can process.
 * Created using pure Node.js (PNG) and macOS sips (JPEG).
 */

/**
 * Valid 10x10 red PNG image (base64)
 * Created using Node.js zlib compression with proper CRC32 checksums.
 * This is a properly encoded PNG that Go's image/png can decode.
 */
const VALID_PNG_BASE64 =
  'iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAIAAAACUFjqAAAAEklEQVR42mP4z8CABzGMSmNDALfKY53W1e90AAAAAElFTkSuQmCC';

/**
 * Valid 10x10 red JPEG image (base64)
 * Created using macOS sips conversion from the PNG above.
 * This is a properly encoded JPEG that Go's image/jpeg can decode.
 */
const VALID_JPEG_BASE64 =
  '/9j/4AAQSkZJRgABAQAASABIAAD/4QBMRXhpZgAATU0AKgAAAAgAAYdpAAQAAAABAAAAGgAAAAAAA6ABAAMAAAABAAEAAKACAAQAAAABAAAACqADAAQAAAABAAAACgAAAAD/7QA4UGhvdG9zaG9wIDMuMAA4QklNBAQAAAAAAAA4QklNBCUAAAAAABDUHYzZjwCyBOmACZjs+EJ+/8AAEQgACgAKAwEiAAIRAQMRAf/EAB8AAAEFAQEBAQEBAAAAAAAAAAABAgMEBQYHCAkKC//EALUQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+v/EAB8BAAMBAQEBAQEBAQEAAAAAAAABAgMEBQYHCAkKC//EALURAAIBAgQEAwQHBQQEAAECdwABAgMRBAUhMQYSQVEHYXETIjKBCBRCkaGxwQkjM1LwFWJy0QoWJDThJfEXGBkaJicoKSo1Njc4OTpDREVGR0hJSlNUVVZXWFlaY2RlZmdoaWpzdHV2d3h5eoKDhIWGh4iJipKTlJWWl5iZmqKjpKWmp6ipqrKztLW2t7i5usLDxMXGx8jJytLT1NXW19jZ2uLj5OXm5+jp6vLz9PX29/j5+v/bAEMAAgICAgICAwICAwUDAwMFBgUFBQUGCAYGBgYGCAoICAgICAgKCgoKCgoKCgwMDAwMDA4ODg4ODw8PDw8PDw8PD//bAEMBAgICBAQEBwQEBxALCQsQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEP/dAAQAAf/aAAwDAQACEQMRAD8A+L6KKK/lM/38P//Z';

/**
 * Get a valid PNG buffer for testing
 */
export function getValidPngBuffer(): Buffer {
  return Buffer.from(VALID_PNG_BASE64, 'base64');
}

/**
 * Get a valid JPEG buffer for testing
 */
export function getValidJpegBuffer(): Buffer {
  return Buffer.from(VALID_JPEG_BASE64, 'base64');
}

/**
 * Get an invalid file buffer for testing error cases
 */
export function getInvalidFileBuffer(): Buffer {
  return Buffer.from('This is not an image file', 'utf-8');
}

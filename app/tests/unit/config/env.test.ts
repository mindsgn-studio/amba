describe('env configuration', () => {
  const original = process.env.EXPO_PUBLIC_API_URL;

  afterEach(() => {
    if (original === undefined) {
      delete process.env.EXPO_PUBLIC_API_URL;
    } else {
      process.env.EXPO_PUBLIC_API_URL = original;
    }
    jest.resetModules();
  });

  function loadEnv(): typeof import('@/config/env') {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    return require('@/config/env') as typeof import('@/config/env');
  }

  it('falls back to localhost when EXPO_PUBLIC_API_URL is unset', () => {
    delete process.env.EXPO_PUBLIC_API_URL;
    jest.resetModules();
    const envModule = loadEnv();
    expect(envModule.API_BASE_URL).toBe('http://localhost:8000');
  });

  it('uses the configured URL (Android emulator)', () => {
    process.env.EXPO_PUBLIC_API_URL = 'http://10.0.2.2:8000';
    jest.resetModules();
    const envModule = loadEnv();
    expect(envModule.API_BASE_URL).toBe('http://10.0.2.2:8000');
  });

  it('strips trailing slashes', () => {
    process.env.EXPO_PUBLIC_API_URL = 'http://192.168.1.100:8000/';
    jest.resetModules();
    const envModule = loadEnv();
    expect(envModule.API_BASE_URL).toBe('http://192.168.1.100:8000');
  });

  it('detects supported upload extensions', () => {
    process.env.EXPO_PUBLIC_API_URL ??= 'http://localhost:8000';
    const { isSupportedUploadFile, getFileExtension } = loadEnv();
    expect(isSupportedUploadFile('book.epub')).toBe(true);
    expect(isSupportedUploadFile('BOOK.PDF')).toBe(true);
    expect(isSupportedUploadFile('notes.txt')).toBe(true);
    expect(isSupportedUploadFile('photo.jpg')).toBe(false);
    expect(isSupportedUploadFile('noextension')).toBe(false);
    expect(getFileExtension('a.b.epub')).toBe('epub');
  });
});

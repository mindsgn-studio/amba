import { resolveApiUrl, apiUrl } from '@/utils/urls';

describe('resolveApiUrl', () => {
  it('prefixes API file paths with the base URL', () => {
    expect(resolveApiUrl('/api/files/books/abc123/chapters/001/chunk_001.mp3')).toBe(
      'http://localhost:8000/api/files/books/abc123/chapters/001/chunk_001.mp3',
    );
  });

  it('normalizes paths missing a leading slash', () => {
    expect(resolveApiUrl('api/jobs/job1')).toBe(
      'http://localhost:8000/api/jobs/job1',
    );
  });

  it('passes absolute URLs through unchanged', () => {
    expect(resolveApiUrl('https://cdn.example.com/audio.mp3')).toBe(
      'https://cdn.example.com/audio.mp3',
    );
    expect(resolveApiUrl('http://10.0.2.2:8000/api/files/x.json')).toBe(
      'http://10.0.2.2:8000/api/files/x.json',
    );
  });
});

describe('apiUrl', () => {
  it('builds endpoint URLs', () => {
    expect(apiUrl('/health')).toBe('http://localhost:8000/health');
    expect(apiUrl(`/api/books/${'abc'}`)).toBe(
      'http://localhost:8000/api/books/abc',
    );
  });
});

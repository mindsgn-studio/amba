declare module 'whisper.rn' {
  export function initWhisper(args: {
    filePath: string;
  }): Promise<{
    transcribe: (
      filePath: string,
      options: { language: string },
    ) => {
      promise: Promise<{ result: string }>;
    };
  }>;
}

// PM2 process definitions for the Read-Along Audiobook System.
// Usage: pm2 start ecosystem.config.js   (from the read-along directory)
module.exports = {
  apps: [
    {
      name: "audiobook-api",
      cwd: __dirname,
      script: ".venv/bin/uvicorn",
      args: "app.api.main:app --host 0.0.0.0 --port 8000",
      interpreter: "none",
      autorestart: true,
      max_restarts: 10,
      restart_delay: 3000,
      max_memory_restart: "1500M",
      env: {
        APP_ENV: "production",
        LOG_LEVEL: "INFO",
      },
    },
    {
      name: "audiobook-worker",
      cwd: __dirname,
      script: ".venv/bin/python",
      args: "-m app.worker.main",
      interpreter: "none",
      autorestart: true,
      max_restarts: 20,
      restart_delay: 3000,
      max_memory_restart: "3000M", // worker holds the TTS model in memory
      env: {
        APP_ENV: "production",
        LOG_LEVEL: "INFO",
      },
    },
  ],
};

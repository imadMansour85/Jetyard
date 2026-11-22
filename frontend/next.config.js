/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // NEXT_OUTPUT=export builds a static site (frontend/out) that the backend
  // serves in the single Docker image. Without it, `next start` works as
  // before for the separate frontend container.
  ...(process.env.NEXT_OUTPUT === 'export' ? { output: 'export' } : {}),
}

module.exports = nextConfig

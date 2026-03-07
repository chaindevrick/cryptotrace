# Stage 1: Dependencies
FROM node:20-alpine AS deps
WORKDIR /app
# 使用萬用字元確保 package.json 與 package-lock.json (如果有的話) 都會被複製
COPY package*.json ./
# 改用 npm install，這樣就算沒有 lock 檔也能自動生成並安裝
RUN npm install

# Stage 2: Builder
FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

# Stage 3: Runner
FROM node:20-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production

# Next.js standalone mode 會自動打包所需的 node_modules
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/export ./
COPY --from=builder /app/.next/static ./.next/static

EXPOSE 3000
ENV PORT=3000

CMD ["node", "server.js"]
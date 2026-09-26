import type { APIRoute } from "astro";
import { readFile } from "node:fs/promises";
import { join } from "node:path";

export const GET: APIRoute = async () => {
  const script = await readFile(join(process.cwd(), "..", "scripts", "install.ps1"), "utf8");
  return new Response(script, {
    headers: {
      "content-type": "text/plain; charset=utf-8",
      "cache-control": "public, max-age=300"
    }
  });
};

import { expect, test } from "@playwright/test";

const runServerlessSmoke = process.env.PLAYWRIGHT_SERVERLESS_SMOKE === "1";

test.describe("serverless deployed smoke", () => {
  test.skip(
    !runServerlessSmoke,
    "Set PLAYWRIGHT_SERVERLESS_SMOKE=1 to run against a deployed CloudFront URL.",
  );

  test("serves the app shell and public API through the same origin", async ({
    page,
    request,
  }) => {
    const protectedDataCalls: string[] = [];

    const healthResponse = await request.get("/api/health");
    expect(healthResponse.status()).toBe(200);
    expect(await healthResponse.json()).toEqual({ message: "ok" });

    const apiRootResponse = await request.get("/api");
    expect(apiRootResponse.status()).toBe(404);
    expect(
      (apiRootResponse.headers()["content-type"] ?? "").toLowerCase(),
    ).not.toContain("text/html");

    const bootstrapResponse = await request.get("/api/bootstrap", {
      headers: {
        "Accept-Language": "ko-KR,ko;q=0.9,en;q=0.8",
      },
    });
    expect(bootstrapResponse.status()).toBe(200);
    expect(await bootstrapResponse.json()).toEqual(
      expect.objectContaining({ authenticated: false }),
    );

    page.on("request", (request) => {
      const path = new URL(request.url()).pathname;
      if (
        /^\/api\/(?:months|goals)(?:\/|$)|^\/api\/(?:checks|memos)(?:\/|$)/.test(
          path,
        )
      ) {
        protectedDataCalls.push(`${request.method()} ${path}`);
      }
    });

    await page.goto("/");

    await expect(page.getByText("Monthly Goal Tracker").first()).toBeVisible();
    await expect(
      page
        .getByText(/체험 모드|미리보기 모드|Guest mode|Preview mode/)
        .first(),
    ).toBeVisible();

    const monthValue = await page
      .getByLabel(/기록할 월|Month to track/)
      .inputValue();
    const firstDayLabel = `${monthValue.slice(5, 7)}.01`;
    const memoInput = page.getByLabel(
      new RegExp(`${escapeRegExp(firstDayLabel)} (메모|memo)`, "i"),
    );
    await memoInput.fill("serverless smoke preview memo");
    await memoInput.blur();

    await expect(
      page
        .getByRole("status")
        .filter({ hasText: /저장하려면 로그인해 주세요|Log in to save/ }),
    ).toBeVisible();

    const goalTitle = "serverless smoke preview goal";
    await page.getByRole("button", { name: /목표 추가|Add goal/ }).click();
    const goalStartDate = await page
      .getByLabel(/새 목표 시작일|New goal start date/)
      .inputValue();
    await page
      .getByLabel(/새 목표 제목|New goal title/)
      .fill(goalTitle);
    await page.getByRole("button", { name: /목표 저장|Save goal/ }).click();
    await expect(
      page.getByRole("complementary").getByText(goalTitle),
    ).toBeVisible();

    const goalCheckButton = page.getByRole("button", {
      name: new RegExp(
        `${escapeRegExp(shortDateLabel(goalStartDate))} ${escapeRegExp(
          goalTitle,
        )} (완료|completed)`,
        "i",
      ),
    });
    await expect(goalCheckButton).toBeEnabled();
    await goalCheckButton.click();
    await expect(goalCheckButton).toHaveAttribute("aria-pressed", "true");

    await page.getByRole("button", { name: /^(로그인|Log in)$/ }).click();
    await expect(
      page.getByRole("dialog", { name: /월간 목표 트래커|Monthly Goal Tracker/ }),
    ).toBeVisible();

    expect(protectedDataCalls).toEqual([]);
  });
});

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function shortDateLabel(value: string): string {
  return value.slice(5).replace("-", ".");
}

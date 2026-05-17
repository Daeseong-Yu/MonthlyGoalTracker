import { expect, test } from "@playwright/test";

const protectedDataPathPattern =
  /^\/api\/(?:months|goals)(?:\/|$)|^\/api\/(?:checks|memos)(?:\/|$)/;

test("anonymous preview keeps changes local and opens login flow", async ({
  page,
}) => {
  const protectedDataCalls: string[] = [];

  await page.route("**/api/bootstrap", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        authenticated: false,
        locale: "ko",
        user: null,
        csrfToken: null,
      }),
    });
  });

  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    if (protectedDataPathPattern.test(path)) {
      protectedDataCalls.push(`${request.method()} ${path}`);
    }
  });

  await page.goto("/");

  await expect(
    page.getByRole("heading", { name: "월간 목표 트래커" }).first(),
  ).toBeVisible();
  await expect(page.getByText("체험 모드")).toBeVisible();
  await expect(page.getByText("저장하려면 로그인해야 합니다.")).toBeVisible();

  const monthValue = await page.getByLabel("기록할 월").inputValue();
  const firstDayMemoLabel = `${monthValue.slice(5, 7)}.01 메모`;
  const memoInput = page.getByLabel(firstDayMemoLabel);
  await memoInput.fill("자동 smoke preview 메모");
  await memoInput.blur();

  await expect(page.getByText("저장하려면 로그인해 주세요")).toBeVisible();
  await expect(
    page.getByRole("button", { name: "로그인하고 저장하기" }),
  ).toHaveCount(0);

  await page.getByRole("button", { name: "목표 추가" }).click();
  await page.getByLabel("새 목표 제목").fill("자동 smoke 목표");
  await page.getByRole("button", { name: "목표 저장" }).click();
  await expect(
    page.getByRole("complementary").getByText("자동 smoke 목표"),
  ).toBeVisible();

  await page.getByRole("button", { name: "목표 추가" }).click();
  await page.getByLabel("새 목표 제목").fill("자동 smoke 두번째 목표");
  await page.getByRole("button", { name: "목표 저장" }).click();
  await expect(
    page.getByRole("complementary").getByText("자동 smoke 두번째 목표"),
  ).toBeVisible();

  await page.getByRole("button", { name: "자동 smoke 두번째 목표 수정" }).click();
  await page
    .getByLabel("자동 smoke 두번째 목표 제목 수정")
    .fill("자동 smoke 두번째 목표 수정됨");
  await page.getByRole("button", { name: "자동 smoke 두번째 목표 저장" }).click();

  await expect(
    page.getByRole("complementary").getByText("자동 smoke 목표", {
      exact: true,
    }),
  ).toBeVisible();
  await expect(
    page.getByRole("complementary").getByText("자동 smoke 두번째 목표 수정됨"),
  ).toBeVisible();

  await page.getByRole("button", { name: "로그인" }).click();
  await expect(
    page.getByRole("dialog", { name: "월간 목표 트래커" }),
  ).toBeVisible();

  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toBeHidden();
  expect(protectedDataCalls).toEqual([]);
});

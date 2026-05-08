import { act, type ReactElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import { vi } from "vitest";

const roots: Root[] = [];

export async function cleanupAppTest() {
  await act(async () => {
    roots.forEach((root) => root.unmount());
  });
  roots.length = 0;
  document.body.innerHTML = "";
  vi.unstubAllGlobals();
}

export function renderApp(element: ReactElement) {
  const container = document.createElement("div");
  document.body.append(container);

  const root = createRoot(container);
  roots.push(root);

  act(() => {
    root.render(element);
  });

  return container;
}

export function stubFetch(
  handler: (
    input: RequestInfo | URL,
    init?: RequestInit,
  ) => Response | Promise<Response>,
) {
  const fetchMock = vi.fn<typeof fetch>().mockImplementation(async (input, init) =>
    handler(input, init),
  );
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

export async function waitForText(text: string) {
  await waitFor(() => document.body.textContent?.includes(text) === true);
}

export async function waitFor(assertion: () => boolean) {
  const start = Date.now();

  while (Date.now() - start < 1000) {
    if (assertion()) {
      return;
    }

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 10));
    });
  }

  throw new Error("timed out waiting for app state");
}

export function getButton(label: string) {
  const button = queryButton(label);

  if (!button) {
    throw new Error(`expected button with aria-label ${label}`);
  }

  return button;
}

export async function clickButton(label: string) {
  await act(async () => {
    getButton(label).dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
}

export async function resolvePending(resolve: (() => void) | null) {
  await act(async () => {
    resolve?.();
  });
}

export function queryButton(label: string) {
  return document.querySelector<HTMLButtonElement>(
    `button[aria-label="${label}"]`,
  );
}

export function getInput(label: string) {
  const input = document.querySelector<HTMLInputElement>(
    `input[aria-label="${label}"]`,
  );

  if (!input) {
    throw new Error(`expected input with aria-label ${label}`);
  }

  return input;
}

export async function setInputValue(label: string, value: string) {
  const input = getInput(label);
  const valueSetter = Object.getOwnPropertyDescriptor(
    HTMLInputElement.prototype,
    "value",
  )?.set;

  await act(async () => {
    valueSetter?.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}

export async function blurInput(label: string) {
  await act(async () => {
    getInput(label).dispatchEvent(new FocusEvent("focusout", { bubbles: true }));
  });
}

export function getHeading(text: string) {
  const heading = Array.from(document.querySelectorAll("h2")).find(
    (item) => item.textContent === text,
  );

  if (!heading) {
    throw new Error(`expected heading ${text}`);
  }

  return heading;
}

export function precedes(first: Element, second: Element) {
  return Boolean(
    first.compareDocumentPosition(second) & Node.DOCUMENT_POSITION_FOLLOWING,
  );
}

export function getDailyRecordTable() {
  const table = document.querySelector("table");

  if (!table) {
    throw new Error("expected daily record table");
  }

  return table;
}

export function getDashboardLayout() {
  const layout = Array.from(document.querySelectorAll("section")).find(
    (section) => section.className.includes("xl:grid-cols-"),
  );

  if (!layout) {
    throw new Error("expected dashboard layout");
  }

  return layout;
}

export function tableHeaderCells() {
  return Array.from(getDailyRecordTable().querySelectorAll("thead th"));
}

export function tableHeaders() {
  return tableHeaderCells().map((header) => header.textContent?.trim() ?? "");
}

export function getWeekdayLabel(shortDay: string) {
  const dateCell = Array.from(
    getDailyRecordTable().querySelectorAll("tbody th"),
  ).find((cell) => cell.textContent?.includes(shortDay));

  if (!dateCell) {
    throw new Error(`expected date cell ${shortDay}`);
  }

  const weekdayLabel = dateCell.querySelector("span:nth-child(2)");

  if (!weekdayLabel) {
    throw new Error(`expected weekday label for ${shortDay}`);
  }

  return weekdayLabel;
}

export function hasInputValue(value: string) {
  return Array.from(document.querySelectorAll("input")).some(
    (input) => input.value === value,
  );
}

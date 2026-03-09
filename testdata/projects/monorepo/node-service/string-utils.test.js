import { expect, test } from "vitest";
import { capitalize, reverse } from "./string-utils.js";

test("capitalize", () => {
  expect(capitalize("hello")).toBe("Hello");
  expect(capitalize("")).toBe("");
});

test("reverse", () => {
  expect(reverse("abc")).toBe("cba");
});

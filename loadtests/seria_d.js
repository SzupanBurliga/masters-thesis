// Seria D: nadmiernie złożone ładunki. VARIANT i LEVEL wybierają wariant ataku i jego poziom.
//  graphql: depth | alias | limit      rest: json-depth | array      grpc: array
import http from "k6/http";
import { WARMUP, DURATION, COOLDOWN, secs } from "./lib/common.js";
import { IFACE, BASE, headers, legitToken, attackerToken, deepQuery, aliasQuery, bigLimitQuery, nestedJson,
         newOrderBody, grpcClient, grpcConnect, grpcMeta } from "./lib/common.js";
import { doOp } from "./lib/ops.js";

const VARIANT = __ENV.VARIANT || "depth";
const LEVEL = parseInt(__ENV.LEVEL || "6");
const BG_RATE = parseInt(__ENV.BG_RATE || "50");
const ATTACK_RATE = parseInt(__ENV.ATTACK_RATE || "10");
const W = secs(WARMUP), D = secs(DURATION), C = secs(COOLDOWN);

export const options = {
  scenarios: {
    legit: { executor: "constant-arrival-rate", exec: "legit", rate: BG_RATE, timeUnit: "1s", duration: `${W + D + C}s`, preAllocatedVUs: 100, maxVUs: 1000, tags: { who: "legit" } },
    complex: { executor: "constant-arrival-rate", exec: "complex", rate: ATTACK_RATE, timeUnit: "1s", duration: `${D}s`, startTime: `${W}s`, preAllocatedVUs: 50, maxVUs: 500, tags: { who: "attacker" } },
  },
  insecureSkipTLSVerify: true,
  summaryTrendStats: ["avg", "med", "p(90)", "p(95)", "p(99)", "max"],
  thresholds: { "http_req_duration{who:legit}": ["max>=0"], "http_req_duration{who:attacker}": ["max>=0"],
                "grpc_req_duration{who:legit}": ["max>=0"], "grpc_req_duration{who:attacker}": ["max>=0"] },
};

export function legit() { doOp("O1", legitToken(), { who: "legit" }); }

export function complex() {
  const tags = { who: "attacker" };
  const token = attackerToken();
  const id = 1 + ((__VU * 104729 + __ITER * 7919) % 10000);
  if (IFACE === "graphql") {
    const q = VARIANT === "alias" ? aliasQuery(LEVEL) : VARIANT === "limit" ? bigLimitQuery(LEVEL) : deepQuery(LEVEL);
    http.post(`${BASE}/graphql`, JSON.stringify({ query: q, variables: { id } }), { headers: headers(token), tags });
  } else if (IFACE === "rest") {
    const body = VARIANT === "array" ? JSON.stringify(newOrderBody(id, LEVEL)) : nestedJson(id, LEVEL);
    http.post(`${BASE}/orders`, body, { headers: headers(token), tags });
  } else {
    grpcConnect();
    const items = newOrderBody(id, LEVEL).items.map((i) => ({ product_id: i.productId, quantity: i.quantity }));
    grpcClient.invoke("shop.v1.Shop/CreateOrder", { user_id: id, items }, { ...grpcMeta(token), tags });
  }
}

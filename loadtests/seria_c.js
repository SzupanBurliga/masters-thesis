// Seria C: nadmiarowe ładunki (SIZE = rozmiar wiadomości na drucie, w bajtach) + legalne tło (O1).
// Ten sam ładunek logiczny (zamówienie z N pozycjami) we wszystkich interfejsach, patrz lib/common.js.
import http from "k6/http";
import { WARMUP, DURATION, COOLDOWN, secs } from "./lib/common.js";
import { check } from "k6";
import { IFACE, BASE, OP, headers, legitToken, attackerToken, bigOrderRest, bigOrderGraphql, bigOrderGrpc,
         grpcClient, grpcConnect, grpcMeta } from "./lib/common.js";
import { doOp } from "./lib/ops.js";

const SIZE = parseInt(__ENV.SIZE || "10240");
const BG_RATE = parseInt(__ENV.BG_RATE || "50");
const BIG_RATE = parseInt(__ENV.BIG_RATE || "20");
const W = secs(WARMUP), D = secs(DURATION), C = secs(COOLDOWN);

export const options = {
  scenarios: {
    legit: { executor: "constant-arrival-rate", exec: "legit", rate: BG_RATE, timeUnit: "1s", duration: `${W + D + C}s`, preAllocatedVUs: 100, maxVUs: 1000, tags: { who: "legit" } },
    big: { executor: "constant-arrival-rate", exec: "big", rate: BIG_RATE, timeUnit: "1s", duration: `${D}s`, startTime: `${W}s`, preAllocatedVUs: 50, maxVUs: 500, tags: { who: "attacker" } },
  },
  insecureSkipTLSVerify: true,
  summaryTrendStats: ["avg", "med", "p(90)", "p(95)", "p(99)", "max"],
  thresholds: { "http_req_duration{who:legit}": ["max>=0"], "http_req_duration{who:attacker}": ["max>=0"],
                "grpc_req_duration{who:legit}": ["max>=0"], "grpc_req_duration{who:attacker}": ["max>=0"] },
};

export function legit() { doOp("O1", legitToken(), { who: "legit" }); }

export function big() {
  const tags = { who: "attacker" };
  const token = attackerToken();
  if (IFACE === "rest") {
    http.post(`${BASE}/orders`, bigOrderRest(SIZE), { headers: headers(token), tags });
  } else if (IFACE === "graphql") {
    http.post(`${BASE}/graphql`, bigOrderGraphql(SIZE), { headers: headers(token), tags });
  } else {
    grpcConnect();
    grpcClient.invoke("shop.v1.Shop/CreateOrder", bigOrderGrpc(SIZE), { ...grpcMeta(token), tags });
  }
}

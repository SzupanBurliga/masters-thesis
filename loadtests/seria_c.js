// Seria C: nadmiarowe ładunki + legalne tło (O1). SIZE = nominalny rozmiar zamówienia w JSON (bajty); wyznacza liczbę
// pozycji N, wspólną dla wszystkich interfejsów (ta sama praca serwera). Rzeczywisty rozmiar na drucie różni się
// formatem i jest zapisywany w metryce attack_payload_bytes (patrz lib/common.js).
import http from "k6/http";
import { WARMUP, DURATION, COOLDOWN, secs } from "./lib/common.js";
import { IFACE, BASE, OP, headers, legitToken, attackerToken, bigOrderRest, bigOrderGraphql, bigOrderGrpc,
         grpcClient, grpcConnect, grpcMeta, itemsForSize, protoSizeCreateOrder,
         attackPayloadBytes, attackPayloadItems } from "./lib/common.js";
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

const N_ITEMS = itemsForSize(SIZE);
let grpcBytes = null; // rozmiar wiadomości gRPC liczony raz na VU (ładunek jest stały)

export function big() {
  const tags = { who: "attacker" };
  const token = attackerToken();
  attackPayloadItems.add(N_ITEMS);
  if (IFACE === "rest") {
    const body = bigOrderRest(SIZE);
    attackPayloadBytes.add(body.length); // ASCII: długość = bajty
    http.post(`${BASE}/orders`, body, { headers: headers(token), tags });
  } else if (IFACE === "graphql") {
    const body = bigOrderGraphql(SIZE);
    attackPayloadBytes.add(body.length);
    http.post(`${BASE}/graphql`, body, { headers: headers(token), tags });
  } else {
    grpcConnect();
    const msg = bigOrderGrpc(SIZE);
    if (grpcBytes === null) grpcBytes = protoSizeCreateOrder(msg);
    attackPayloadBytes.add(grpcBytes);
    grpcClient.invoke("shop.v1.Shop/CreateOrder", msg, { ...grpcMeta(token), tags });
  }
}

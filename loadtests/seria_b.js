// Seria B: atak wolumetryczny. Legalni klienci (wiele tożsamości) + napastnik (jedna tożsamość).
// Oś czasu: LEGIT_ONLY tylko legalni, ATTACK_DUR atak, RECOVER obserwacja powrotu.
import { OP, AUTH, legitToken, attackerToken } from "./lib/common.js";
import { doOp } from "./lib/ops.js";

const LEGIT_RATE = parseInt(__ENV.LEGIT_RATE || "50");
const ATTACK_RATE = parseInt(__ENV.ATTACK_RATE || "500");
const LEGIT_ONLY = parseInt(__ENV.LEGIT_ONLY || "120");
const ATTACK_DUR = parseInt(__ENV.ATTACK_DUR || "300");
const RECOVER = parseInt(__ENV.RECOVER || "180");

export const options = {
  scenarios: {
    legit: {
      executor: "constant-arrival-rate", exec: "legit",
      rate: LEGIT_RATE, timeUnit: "1s", duration: `${LEGIT_ONLY + ATTACK_DUR + RECOVER}s`,
      preAllocatedVUs: 100, maxVUs: 1000, tags: { who: "legit" },
    },
    attacker: {
      executor: "constant-arrival-rate", exec: "attacker",
      rate: ATTACK_RATE, timeUnit: "1s", duration: `${ATTACK_DUR}s`, startTime: `${LEGIT_ONLY}s`,
      preAllocatedVUs: 300, maxVUs: 5000, tags: { who: "attacker" },
    },
  },
  insecureSkipTLSVerify: true,
  summaryTrendStats: ["avg", "med", "p(90)", "p(95)", "p(99)", "max"],
  thresholds: { "http_req_duration{who:legit}": ["max>=0"], "http_req_duration{who:attacker}": ["max>=0"],
                "grpc_req_duration{who:legit}": ["max>=0"], "grpc_req_duration{who:attacker}": ["max>=0"] },
};

export function legit() { doOp(OP, legitToken(), { who: "legit" }); }
export function attacker() { doOp(OP, attackerToken(), { who: "attacker" }); }

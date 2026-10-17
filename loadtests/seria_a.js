// Seria A: narzut przy ruchu normalnym. MODE=ramp (A1: punkt nasycenia) | const (A2: obciążenie stałe).
import { RATE, WARMUP, DURATION, COOLDOWN, OP, AUTH, legitToken, secs } from "./lib/common.js";
import { doOp } from "./lib/ops.js";

const MODE = __ENV.MODE || "const";
const MAXRATE = parseInt(__ENV.MAXRATE || "2000");

const scenarios = MODE === "ramp"
  ? {
      ramp: {
        executor: "ramping-arrival-rate", startRate: 10, timeUnit: "1s",
        preAllocatedVUs: 200, maxVUs: 3000,
        stages: [{ target: MAXRATE, duration: __ENV.RAMP || "5m" }],
      },
    }
  : {
      warmup: { executor: "constant-arrival-rate", rate: RATE, timeUnit: "1s", duration: WARMUP, preAllocatedVUs: 100, maxVUs: 2000, tags: { phase: "warmup" } },
      measure: { executor: "constant-arrival-rate", rate: RATE, timeUnit: "1s", duration: DURATION, startTime: WARMUP, preAllocatedVUs: 100, maxVUs: 2000, tags: { phase: "measure" } },
      cooldown: { executor: "constant-arrival-rate", rate: RATE, timeUnit: "1s", duration: COOLDOWN, startTime: `${secs(WARMUP) + secs(DURATION)}s`, preAllocatedVUs: 100, maxVUs: 2000, tags: { phase: "cooldown" } },
    };

export const options = {
  scenarios,
  insecureSkipTLSVerify: true,
  summaryTrendStats: ["avg", "med", "p(90)", "p(95)", "p(99)", "max"],
  // percentyle tylko z fazy pomiarowej
  thresholds: { "http_req_duration{phase:measure}": ["max>=0"], "grpc_req_duration{phase:measure}": ["max>=0"] },
};

export default function () {
  doOp(OP, legitToken(), {});
}

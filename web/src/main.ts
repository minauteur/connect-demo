import { createClient } from "@connectrpc/connect";
import { createConnectTransport, createGrpcWebTransport } from "@connectrpc/connect-web";
import { DemoService } from "./gen/demo/v1/demo_pb";
import "./styles.css";

type ProtocolMode = "connect" | "grpc-web" | "grpc-gateway";

const form = document.querySelector<HTMLFormElement>("#demo-form");
const output = document.querySelector<HTMLElement>("#response");
const protocolInput = document.querySelector<HTMLSelectElement>("#protocol");
const nameInput = document.querySelector<HTMLInputElement>("#name");

if (!form || !output || !protocolInput || !nameInput) {
  throw new Error("UI did not initialize correctly");
}

const connectClient = createClient(
  DemoService,
  createConnectTransport({
    baseUrl: "http://localhost:8080",
  }),
);

const grpcWebClient = createClient(
  DemoService,
  createGrpcWebTransport({
    baseUrl: "http://localhost:8080",
  }),
);

form.addEventListener("submit", async (event) => {
  event.preventDefault();

  const protocol = protocolInput.value as ProtocolMode;
  const name = nameInput.value.trim();
  output.textContent = "Sending request...";

  try {
    let response: unknown;

    if (protocol === "connect") {
      response = await connectClient.sayHello({ name });
    } else if (protocol === "grpc-web") {
      response = await grpcWebClient.sayHello({ name });
    } else {
      const gatewayResponse = await fetch("http://localhost:8080/v1/hello:echo", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ name }),
      });

      if (!gatewayResponse.ok) {
        const body = await gatewayResponse.text();
        throw new Error(`grpc-gateway error ${gatewayResponse.status}: ${body}`);
      }

      response = await gatewayResponse.json();
    }

    output.textContent = JSON.stringify(response, null, 2);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    output.textContent = `Request failed: ${message}`;
  }
});

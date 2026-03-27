package com.sap.mcp.proxy;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;
import com.sap.mcp.proxy.model.ProxyRequest;
import com.sap.mcp.proxy.model.ProxyResponse;
import com.sap.mcp.proxy.model.RfcCallRequest;
import com.sap.mcp.proxy.model.RfcCallResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.*;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * STDIO-based transport for the JCo sidecar proxy.
 * <p>
 * Instead of running an HTTP server, this transport reads JSON requests
 * from stdin and writes JSON responses to stdout, one per line (newline-delimited JSON).
 * <p>
 * Protocol:
 * <pre>
 * Request (stdin):  {"id":"<id>","type":"proxy|rfc-call|health|pool-status|terminate","request":{...}}
 * Response (stdout): {"id":"<id>","response":{...},"error":null}
 * </pre>
 * <p>
 * The "id" field correlates requests with responses. All logging goes to stderr.
 */
public class StdioTransport {
    private static final Logger logger = LoggerFactory.getLogger(StdioTransport.class);
    private static final Gson gson = new GsonBuilder().create();

    private final JCoConnectionManager connectionManager;
    private final StatefulSessionManager sessionManager;
    private final RestRfcEndpointCaller rfcCaller;
    private final DirectRfcCaller directRfcCaller;

    private final BufferedReader reader;
    private final PrintWriter writer;

    private volatile boolean running = true;

    public StdioTransport(JCoConnectionManager connectionManager) {
        this.connectionManager = connectionManager;
        this.sessionManager = new StatefulSessionManager();
        this.rfcCaller = new RestRfcEndpointCaller(connectionManager);
        this.directRfcCaller = new DirectRfcCaller(connectionManager);
        this.reader = new BufferedReader(new InputStreamReader(System.in));
        // Use stderr-redirected stdout: actual stdout is reserved for protocol messages
        this.writer = new PrintWriter(new BufferedOutputStream(System.out), false);
    }

    /**
     * Run the STDIO transport loop.
     * Reads JSON requests from stdin, dispatches them, and writes responses to stdout.
     */
    public void run() {
        // Signal readiness to the parent process
        writer.println("SIDECAR_READY");
        writer.flush();

        logger.info("STDIO transport ready, waiting for requests on stdin");

        while (running) {
            try {
                String line = reader.readLine();
                if (line == null) {
                    // EOF - parent process closed stdin
                    logger.info("stdin closed, shutting down");
                    break;
                }

                line = line.trim();
                if (line.isEmpty()) {
                    continue;
                }

                handleMessage(line);

            } catch (IOException e) {
                if (running) {
                    logger.error("Error reading from stdin: {}", e.getMessage());
                }
                break;
            }
        }

        shutdown();
    }

    /**
     * Parse and dispatch a single JSON message.
     */
    private void handleMessage(String line) {
        String id = null;
        try {
            JsonObject msg = JsonParser.parseString(line).getAsJsonObject();
            id = msg.has("id") ? msg.get("id").getAsString() : null;
            String type = msg.has("type") ? msg.get("type").getAsString() : "";

            switch (type) {
                case "proxy":
                    handleProxy(id, msg);
                    break;
                case "rfc-call":
                    handleRfcCall(id, msg);
                    break;
                case "health":
                    handleHealth(id);
                    break;
                case "pool-status":
                    handlePoolStatus(id);
                    break;
                case "terminate":
                    handleTerminate(id, msg);
                    break;
                default:
                    sendError(id, "Unknown message type: " + type);
                    break;
            }
        } catch (Exception e) {
            logger.error("Error handling message: {}", e.getMessage(), e);
            sendError(id, "Error: " + e.getMessage());
        }
    }

    /**
     * Handle proxy request (equivalent to /rfc-proxy HTTP endpoint).
     */
    private void handleProxy(String id, JsonObject msg) {
        try {
            ProxyRequest request = gson.fromJson(msg.get("request"), ProxyRequest.class);

            if (request == null || request.getMethod() == null || request.getUri() == null) {
                sendProxyResponse(id, ProxyResponse.error(400, "Invalid request: method and uri are required"));
                return;
            }

            logger.debug("STDIO proxy request: {} {}", request.getMethod(), request.getUri());

            // Check for stateful session
            String sessionType = request.getHeaders() != null
                ? request.getHeaders().get("X-sap-adt-sessiontype") : null;
            boolean isStateful = "stateful".equalsIgnoreCase(sessionType);
            logger.info("[STDIO] {} {} | sessionType={}", request.getMethod(),
                request.getUri().substring(0, Math.min(80, request.getUri().length())), sessionType);

            String sessionId = null;
            if (isStateful) {
                String cookieHeader = request.getHeaders() != null
                    ? request.getHeaders().get("Cookie") : null;
                sessionId = extractSessionId(cookieHeader);
                sessionId = sessionManager.beginSession(connectionManager.getDestination(), sessionId);
            }

            // Inject SAP cookies for session continuity
            if (isStateful && sessionId != null && sessionManager.isStatefulSession(sessionId)) {
                injectSapCookies(request, sessionId);
            }

            // Execute on session thread
            final String finalSessionId = sessionId;
            ProxyResponse response = sessionManager.executeInSession(sessionId, () -> {
                logger.info("[STDIO] Executing RFC on thread: {} (session: {})",
                    Thread.currentThread().getName(), finalSessionId);
                return rfcCaller.execute(request);
            });

            // Store SAP cookies
            if (isStateful && sessionId != null) {
                storeSapCookies(response, sessionId);
                String cookieValue = "sap-contextid=" + sessionId + "; Path=/; HttpOnly";
                response.getHeaders().put("Set-Cookie", cookieValue);
            }

            sendProxyResponse(id, response);

        } catch (Exception e) {
            logger.error("Error in proxy handler: {}", e.getMessage(), e);
            sendProxyResponse(id, ProxyResponse.error(500, "Proxy error: " + e.getMessage()));
        }
    }

    /**
     * Handle direct RFC call (equivalent to /rfc-call HTTP endpoint).
     */
    private void handleRfcCall(String id, JsonObject msg) {
        try {
            RfcCallRequest request = gson.fromJson(msg.get("request"), RfcCallRequest.class);

            if (request == null || request.getFunction() == null || request.getFunction().isEmpty()) {
                sendRfcResponse(id, RfcCallResponse.error("Invalid request: function name is required"));
                return;
            }

            logger.info("[STDIO] RFC call: {}", request.getFunction());

            RfcCallResponse response = directRfcCaller.execute(request);
            sendRfcResponse(id, response);

        } catch (Exception e) {
            logger.error("Error in RFC call handler: {}", e.getMessage(), e);
            sendRfcResponse(id, RfcCallResponse.error("RFC call error: " + e.getMessage()));
        }
    }

    /**
     * Handle health check.
     */
    private void handleHealth(String id) {
        boolean healthy = connectionManager.isHealthy();
        JsonObject resp = new JsonObject();
        resp.addProperty("id", id);
        resp.addProperty("healthy", healthy);
        resp.addProperty("status", healthy ? "OK" : "SAP connection unhealthy");
        sendRaw(resp);
    }

    /**
     * Handle pool status query.
     */
    private void handlePoolStatus(String id) {
        try {
            Map<String, Object> sessionInfo = sessionManager.getSessionInfo();

            Map<String, Object> jcoConfig = new HashMap<>();
            jcoConfig.put("poolCapacity", 5);
            jcoConfig.put("peakLimit", 10);

            Map<String, Object> response = new HashMap<>();
            response.put("sessions", sessionInfo.get("sessions"));
            response.put("sessionTimeout", sessionInfo.get("timeout"));
            response.put("activeSessionCount", sessionInfo.get("count"));
            response.put("jcoPoolConfig", jcoConfig);

            JsonObject resp = new JsonObject();
            resp.addProperty("id", id);
            resp.add("response", gson.toJsonTree(response));
            sendRaw(resp);

        } catch (Exception e) {
            logger.error("Error getting pool status: {}", e.getMessage(), e);
            sendError(id, "Error getting pool status: " + e.getMessage());
        }
    }

    /**
     * Handle session termination.
     */
    private void handleTerminate(String id, JsonObject msg) {
        try {
            JsonObject req = msg.has("request") ? msg.getAsJsonObject("request") : new JsonObject();

            String sessionId = req.has("sessionId") ? req.get("sessionId").getAsString() : null;
            Integer ageThreshold = req.has("ageThresholdSeconds")
                ? req.get("ageThresholdSeconds").getAsInt() : null;
            Boolean force = req.has("force") ? req.get("force").getAsBoolean() : null;

            List<Map<String, Object>> terminated =
                sessionManager.terminateSessions(sessionId, ageThreshold, force);

            Map<String, Object> response = new HashMap<>();
            response.put("terminatedCount", terminated.size());
            response.put("sessions", terminated);

            JsonObject resp = new JsonObject();
            resp.addProperty("id", id);
            resp.add("response", gson.toJsonTree(response));
            sendRaw(resp);

        } catch (Exception e) {
            logger.error("Error terminating sessions: {}", e.getMessage(), e);
            sendError(id, "Error terminating sessions: " + e.getMessage());
        }
    }

    // --- Response helpers ---

    private void sendProxyResponse(String id, ProxyResponse response) {
        JsonObject resp = new JsonObject();
        resp.addProperty("id", id);
        resp.add("response", gson.toJsonTree(response));
        sendRaw(resp);
    }

    private void sendRfcResponse(String id, RfcCallResponse response) {
        JsonObject resp = new JsonObject();
        resp.addProperty("id", id);
        resp.add("response", gson.toJsonTree(response));
        sendRaw(resp);
    }

    private void sendError(String id, String message) {
        JsonObject resp = new JsonObject();
        if (id != null) {
            resp.addProperty("id", id);
        }
        resp.addProperty("error", message);
        sendRaw(resp);
    }

    private synchronized void sendRaw(JsonObject obj) {
        writer.println(gson.toJson(obj));
        writer.flush();
    }

    // --- Session helpers (same as RfcProxyServer) ---

    private void injectSapCookies(ProxyRequest request, String sessionId) {
        String sapCookies = sessionManager.getSapCookies(sessionId);
        if (sapCookies != null) {
            request.getHeaders().put("Cookie", sapCookies);
            logger.info("[STDIO] Injected SAP cookies: {}", sapCookies);
        }
    }

    private void storeSapCookies(ProxyResponse response, String sessionId) {
        String setCookie = response.getHeaders().get("Set-Cookie");
        if (setCookie == null) setCookie = response.getHeaders().get("set-cookie");
        if (setCookie != null && !setCookie.isEmpty()) {
            sessionManager.storeSapCookies(sessionId, setCookie);
        }
    }

    private String extractSessionId(String cookieHeader) {
        if (cookieHeader == null || cookieHeader.isEmpty()) return null;
        for (String cookie : cookieHeader.split(";")) {
            String[] parts = cookie.trim().split("=", 2);
            if (parts.length == 2 && "sap-contextid".equals(parts[0])) {
                return parts[1];
            }
        }
        return null;
    }

    /**
     * Shut down the transport and release resources.
     */
    public void shutdown() {
        running = false;
        sessionManager.shutdown();
        connectionManager.close();
        logger.info("STDIO transport shut down");
    }
}

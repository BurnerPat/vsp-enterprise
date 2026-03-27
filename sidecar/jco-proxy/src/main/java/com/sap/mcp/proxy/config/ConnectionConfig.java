package com.sap.mcp.proxy.config;

import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Properties;

/**
 * Configuration holder for SAP connection settings.
 * Supports both command-line arguments and environment variables.
 *
 * Two modes of operation:
 * <ul>
 *   <li><b>Named parameters:</b> Individual --ashost/--mshost/--user/... flags
 *       (traditional user+password logon)</li>
 *   <li><b>JCo properties:</b> Arbitrary --jco.&lt;property&gt; flags passed directly
 *       to the JCo destination configuration (e.g., SNC/SSO logon).</li>
 * </ul>
 *
 * These two modes are mutually exclusive. If any --jco.* parameter is present,
 * named connection parameters must not be used.
 */
public class ConnectionConfig {
    // Load balancing via message server
    private String msHost;
    private String msServ;
    private String r3Name;
    private String group;

    // Direct application server connection
    private String asHost;
    private String sysnr;

    // Common settings
    private String client;
    private String username;
    private String password;
    private String language;

    // Raw JCo properties (e.g., from --jco.client.snc_mode 1 --jco.client.mshost ...)
    // When populated, these are used instead of the named fields above.
    private final Map<String, String> jcoProperties = new LinkedHashMap<>();

    public ConnectionConfig() {
        this.language = "EN";
    }

    /**
     * Check if raw JCo properties mode is active.
     * When true, connection configuration comes entirely from jcoProperties
     * rather than individual named fields.
     */
    public boolean isJcoPropertiesMode() {
        return !jcoProperties.isEmpty();
    }

    /**
     * Check if this is a direct application server connection (vs load balancing).
     */
    public boolean isDirectConnection() {
        return asHost != null && !asHost.isEmpty();
    }

    /**
     * Create configuration from command-line arguments.
     * Expected format: --key value
     *
     * Supports two mutually exclusive modes:
     * <ul>
     *   <li>Named parameters: --ashost, --user, --client, etc.</li>
     *   <li>JCo properties: --jco.client.snc_mode, --jco.client.mshost, etc.</li>
     * </ul>
     */
    public static ConnectionConfig fromArgs(String[] args) {
        ConnectionConfig config = new ConnectionConfig();
        boolean hasNamedParams = false;
        boolean hasJcoParams = false;

        for (int i = 0; i < args.length; i++) {
            String arg = args[i];

            // JCo property arguments: --jco.<property> <value>
            if (arg.startsWith("--jco.") && i + 1 < args.length) {
                // Strip the leading "--" to get the JCo property key
                // e.g., "--jco.client.snc_mode" -> "jco.client.snc_mode"
                String jcoKey = arg.substring(2);
                config.jcoProperties.put(jcoKey, args[++i]);
                hasJcoParams = true;
                continue;
            }

            switch (arg) {
                // Load balancing settings
                case "--mshost":
                    if (i + 1 < args.length) {
                        config.setMsHost(args[++i]);
                        hasNamedParams = true;
                    }
                    break;
                case "--msserv":
                    if (i + 1 < args.length) {
                        config.setMsServ(args[++i]);
                        hasNamedParams = true;
                    }
                    break;
                case "--r3name":
                    if (i + 1 < args.length) {
                        config.setR3Name(args[++i]);
                        hasNamedParams = true;
                    }
                    break;
                case "--group":
                    if (i + 1 < args.length) {
                        config.setGroup(args[++i]);
                        hasNamedParams = true;
                    }
                    break;
                // Direct app server settings
                case "--ashost":
                    if (i + 1 < args.length) {
                        config.setAsHost(args[++i]);
                        hasNamedParams = true;
                    }
                    break;
                case "--sysnr":
                    if (i + 1 < args.length) {
                        config.setSysnr(args[++i]);
                        hasNamedParams = true;
                    }
                    break;
                // Common settings
                case "--client":
                    if (i + 1 < args.length) {
                        config.setClient(args[++i]);
                        // client is a common param, allowed in both modes
                    }
                    break;
                case "--user":
                    if (i + 1 < args.length) {
                        config.setUsername(args[++i]);
                        // user is a common param, allowed in both modes
                    }
                    break;
                case "--password":
                    if (i + 1 < args.length) {
                        config.setPassword(args[++i]);
                        hasNamedParams = true;
                    }
                    break;
                case "--lang":
                    if (i + 1 < args.length) {
                        config.setLanguage(args[++i]);
                        // language is a common param, allowed in both modes
                    }
                    break;
            }
        }

        // Enforce mutual exclusivity
        if (hasJcoParams && hasNamedParams) {
            throw new IllegalArgumentException(
                "Cannot mix --jco.* properties with named connection parameters "
                + "(--ashost, --mshost, --user, etc.). Use one mode or the other.");
        }

        return config;
    }

    /**
     * Create configuration from environment variables.
     */
    public static ConnectionConfig fromEnvironment() {
        ConnectionConfig config = new ConnectionConfig();

        // Load balancing settings
        config.setMsHost(getEnv("SAP_MSHOST"));
        config.setMsServ(getEnv("SAP_MSSERV"));
        config.setR3Name(getEnv("SAP_R3NAME"));
        config.setGroup(getEnv("SAP_GROUP"));

        // Direct app server settings
        config.setAsHost(getEnv("SAP_ASHOST"));
        config.setSysnr(getEnv("SAP_SYSNR", "00"));

        // Common settings
        config.setClient(getEnv("SAP_CLIENT"));
        config.setUsername(getEnv("SAP_USERNAME"));
        config.setPassword(getEnv("SAP_PASSWORD"));
        config.setLanguage(getEnv("SAP_LANGUAGE", "EN"));

        return config;
    }

    /**
     * Merge command-line args over environment config (args take precedence).
     *
     * If the args config uses JCo properties mode, environment variables are ignored
     * entirely - the JCo properties contain the complete connection configuration.
     */
    public static ConnectionConfig merge(ConnectionConfig envConfig, ConnectionConfig argsConfig) {
        // JCo properties mode: no merging, args contain everything
        if (argsConfig.isJcoPropertiesMode()) {
            return argsConfig;
        }

        ConnectionConfig merged = new ConnectionConfig();

        // Load balancing settings
        merged.setMsHost(coalesce(argsConfig.getMsHost(), envConfig.getMsHost()));
        merged.setMsServ(coalesce(argsConfig.getMsServ(), envConfig.getMsServ()));
        merged.setR3Name(coalesce(argsConfig.getR3Name(), envConfig.getR3Name()));
        merged.setGroup(coalesce(argsConfig.getGroup(), envConfig.getGroup()));

        // Direct app server settings
        merged.setAsHost(coalesce(argsConfig.getAsHost(), envConfig.getAsHost()));
        merged.setSysnr(coalesce(argsConfig.getSysnr(), envConfig.getSysnr()));

        // Common settings
        merged.setClient(coalesce(argsConfig.getClient(), envConfig.getClient()));
        merged.setUsername(coalesce(argsConfig.getUsername(), envConfig.getUsername()));
        merged.setPassword(coalesce(argsConfig.getPassword(), envConfig.getPassword()));
        merged.setLanguage(coalesce(argsConfig.getLanguage(), envConfig.getLanguage()));

        return merged;
    }

    public void validate() {
        // JCo properties mode: properties are passed through to JCo as-is,
        // no validation needed on our side — JCo will reject invalid config.
        if (isJcoPropertiesMode()) {
            if (jcoProperties.isEmpty()) {
                throw new IllegalArgumentException("JCo properties mode active but no properties provided");
            }
            return;
        }

        StringBuilder errors = new StringBuilder();

        // Check for either direct connection or load balancing settings
        boolean hasDirect = !isEmpty(asHost);
        boolean hasLoadBalancing = !isEmpty(msHost);

        if (!hasDirect && !hasLoadBalancing) {
            errors.append("Either asHost (direct) or msHost (load balancing) is required\n");
        }

        if (hasDirect && hasLoadBalancing) {
            errors.append("Cannot specify both asHost (direct) and msHost (load balancing) - choose one mode\n");
        }

        if (hasDirect) {
            // Direct connection validation
            if (isEmpty(sysnr)) errors.append("sysnr is required for direct connection\n");
        } else if (hasLoadBalancing) {
            // Load balancing validation
            if (isEmpty(msServ)) errors.append("msServ is required for load balancing\n");
            if (isEmpty(r3Name)) errors.append("r3Name is required for load balancing\n");
            if (isEmpty(group)) errors.append("group is required for load balancing\n");
        }

        // Common validation
        if (isEmpty(client)) errors.append("client is required\n");
        if (isEmpty(username)) errors.append("username is required\n");
        if (isEmpty(password)) errors.append("password is required\n");

        if (errors.length() > 0) {
            throw new IllegalArgumentException("Invalid configuration:\n" + errors);
        }
    }

    private static String getEnv(String key) {
        return System.getenv(key);
    }

    private static String getEnv(String key, String defaultValue) {
        String value = System.getenv(key);
        return value != null ? value : defaultValue;
    }

    private static String coalesce(String... values) {
        for (String v : values) {
            if (v != null && !v.isEmpty()) return v;
        }
        return null;
    }

    private static boolean isEmpty(String s) {
        return s == null || s.isEmpty();
    }

    /**
     * Get the raw JCo properties map.
     * Only populated when --jco.* arguments were passed.
     */
    public Map<String, String> getJcoProperties() {
        return Collections.unmodifiableMap(jcoProperties);
    }

    /**
     * Convert JCo properties to a java.util.Properties object
     * suitable for passing directly to the JCo DestinationDataProvider.
     */
    public Properties toJcoDestinationProperties() {
        Properties props = new Properties();
        for (Map.Entry<String, String> entry : jcoProperties.entrySet()) {
            props.setProperty(entry.getKey(), entry.getValue());
        }

        // Inject common named params (--client, --user, --lang) into JCo properties
        // if they were specified and not already present in the JCo properties map.
        if (!isEmpty(client) && !props.containsKey("jco.client.client")) {
            props.setProperty("jco.client.client", client);
        }
        if (!isEmpty(username) && !props.containsKey("jco.client.user")) {
            props.setProperty("jco.client.user", username);
        }
        if (!isEmpty(language) && !props.containsKey("jco.client.lang")) {
            props.setProperty("jco.client.lang", language);
        }

        return props;
    }

    // Getters and Setters - Load balancing
    public String getMsHost() { return msHost; }
    public void setMsHost(String msHost) { this.msHost = msHost; }

    public String getMsServ() { return msServ; }
    public void setMsServ(String msServ) { this.msServ = msServ; }

    public String getR3Name() { return r3Name; }
    public void setR3Name(String r3Name) { this.r3Name = r3Name; }

    public String getGroup() { return group; }
    public void setGroup(String group) { this.group = group; }

    // Getters and Setters - Direct connection
    public String getAsHost() { return asHost; }
    public void setAsHost(String asHost) { this.asHost = asHost; }

    public String getSysnr() { return sysnr; }
    public void setSysnr(String sysnr) { this.sysnr = sysnr; }

    // Getters and Setters - Common
    public String getClient() { return client; }
    public void setClient(String client) { this.client = client; }

    public String getUsername() { return username; }
    public void setUsername(String username) { this.username = username; }

    public String getPassword() { return password; }
    public void setPassword(String password) { this.password = password; }

    public String getLanguage() { return language; }
    public void setLanguage(String language) { this.language = language; }

    @Override
    public String toString() {
        if (isJcoPropertiesMode()) {
            StringBuilder sb = new StringBuilder("ConnectionConfig{mode='jcoProperties', properties=[");
            boolean first = true;
            for (Map.Entry<String, String> entry : jcoProperties.entrySet()) {
                if (!first) sb.append(", ");
                first = false;
                // Mask sensitive values
                String key = entry.getKey();
                if (key.contains("passwd") || key.contains("password")) {
                    sb.append(key).append("='***'");
                } else {
                    sb.append(key).append("='").append(entry.getValue()).append("'");
                }
            }
            sb.append("]}");
            return sb.toString();
        }

        StringBuilder sb = new StringBuilder("ConnectionConfig{");
        if (isDirectConnection()) {
            sb.append("mode='direct'")
              .append(", asHost='").append(asHost).append('\'')
              .append(", sysnr='").append(sysnr).append('\'');
        } else {
            sb.append("mode='loadBalancing'")
              .append(", msHost='").append(msHost).append('\'')
              .append(", msServ='").append(msServ).append('\'')
              .append(", r3Name='").append(r3Name).append('\'')
              .append(", group='").append(group).append('\'');
        }
        sb.append(", client='").append(client).append('\'')
          .append(", username='").append(username).append('\'')
          .append(", language='").append(language).append('\'')
          .append('}');
        return sb.toString();
    }
}

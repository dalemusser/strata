Title: Strata requirements  
Author: Dale Musser  
Date: Monday, September 9, 2024

The **requirements** for the Strata server, including the **core server functionality** and the **Authentication UI and Flow**:

---

### **Core Features**

1. **Automatic TLS/SSL Certificates via Let's Encrypt**:
   - Automatic certificate management using Let's Encrypt.
   - HTTP server on port 80 for Let's Encrypt validation.
   - Fallback to HTTP for numeric IP addresses, localhost, or non-routable IPs (since Let's Encrypt requires a domain name for SSL).

2. **Manual TLS/SSL Certificates**:
   - Support for manually provided certificates via `CertFile` and `KeyFile` fields in the config.
   - If certificates are present, they are used instead of Let's Encrypt.
   - If certificates are missing, Let's Encrypt should be used automatically (unless disabled).

3. **Dynamic TLS Configuration**:
   - Dynamically configure the TLS setup based on the environment (i.e., Let's Encrypt or manual certificates).
   - Ability to choose between automatic or manual TLS certificates via config settings.

4. **HTTP-to-HTTPS Redirection**:
   - Redirect HTTP traffic on port 80 to HTTPS on port 443 automatically.

5. **Support for HTTP when needed**:
   - Automatically use HTTP instead of HTTPS on localhost, non-routable addresses, and numeric IP addresses.

6. **Graceful Shutdown**:
   - Support for graceful shutdown when the server receives termination signals (e.g., SIGINT or SIGTERM).
   - Configurable shutdown timeout.

---

### **Routing and Static File Serving**

7. **Serving Static Files**:
   - Serve static files from a configurable `static/` directory (default: `./static/`).
   - Files in the `static/` directory should be accessible directly from the root URL, e.g., `https://example.com/image1.png` (not `https://example.com/static/image1.png`).
   - The URL structure maps directly to the file system, with no `/static/` prefix in the URL path unless explicitly configured.

8. **Dynamic Handling of Index File**:
   - If an `index.html` file exists in the static directory, serve it when accessing the root (`/`).
   - If `index.html` does not exist, dynamically serve a login page or other content based on the server state.

9. **Optional CDN Support**:
   - Allow serving static files from a CloudFront CDN when configured.
   - Fallback to local static file serving when the CDN is not configured or available.

---

### **Authentication UI and Flow**

10. **Login Flow**:
    - The server should present a **login page** if the user is not authenticated.
    - If the user is authenticated (via JWT), the **dashboard** or authenticated content should be shown instead of the login page.
    - Use **HTMx** to dynamically load the login and dashboard components into the base template without reloading the page.

11. **Authentication Provider Menu**:
    - The login page should present a **menu of authentication providers** (e.g., Google, ClassLink, etc.).
    - The user selects the desired provider from the menu, and based on the selection, the corresponding login form is dynamically displayed using **HTMx**.

12. **Use of HTMx for Dynamic Login Form**:
    - The login form should **dynamically update** based on the selected provider without a full page reload.
    - HTMx should handle this declaratively, dynamically updating the login form with the appropriate provider.

13. **JWT-Based Authentication**:
    - Upon successful authentication, the server should issue a **JWT** to the client.
    - The JWT should be stored **client-side** (e.g., in `localStorage`), and used in subsequent requests to authenticated endpoints.
  
14. **Multiple Authentication Providers (via Goth)**:
    - Support multiple authentication providers such as:
      - Google, Discord, GitHub, Facebook, Twitter, ClassLink, LinkedIn, Microsoft, Bitbucket, Dropbox, Twitch.
    - Dynamically initialize the selected provider and handle the OAuth login flow via Goth.

15. **Flexible Login Flow**:
    - Provide a **non-authenticated experience** where users can browse certain content (e.g., a public-facing homepage or static files) without logging in.
    - If authentication is required, gracefully trigger the login flow via **HTMx redirects** when users attempt to access protected content.

16. **Feedback for Incorrect Login**:
    - Provide **feedback** (e.g., incorrect credentials, provider issues) via HTMx or standard HTML when the login fails.

---

### **Configuration Management**

17. **Flexible Configuration**:
    - Support configuration through multiple sources:
      - **Config file**: YAML, JSON, or TOML.
      - **Environment variables**.
      - **Command-line arguments**.
    - A command-line flag should allow specifying the path to a configuration file.
    - Environment variables and command-line parameters should override settings in the configuration file.

18. **Configurable Settings**:
    - Server host, port, certificate files, CloudFront URL, log file path, static directory, and more should be configurable.
    - Option to configure **CloudWatch** for logging:
      - CloudWatch Log Group and Log Stream, AWS Region, etc.
    - Option to configure **CloudFront** for static file serving.

19. **Fallback and Validation for Configuration**:
    - Default values for missing config parameters.
    - Handle cases where certain config values are missing or invalid.
    - Proper validation and error reporting for configuration issues (e.g., invalid certificate paths).
    - Gracefully ignore extra or unused configuration fields.

---

### **Logging**

20. **Local and Remote Logging**:
    - Log locally to a file (path configurable).
    - Support remote logging to services such as:
      - **Amazon CloudWatch** (log group, log stream, AWS region).
    - Ability to use multiple logging services concurrently.
    - Logging should not fail if certain services (e.g., CloudWatch) are not configured or available.

---

### **Compression**

21. **Gzip and Brotli Compression**:
    - Serve files with Gzip or Brotli compression.
    - Check for pre-compressed files (e.g., `.br`, `.gz`) and serve them if they exist.
    - Dynamically compress responses with Gzip or Brotli if no pre-compressed file is available.

---

### **User Interface**

22. **HTMx and Flowbite for UI Components**:
    - Use HTMx to handle dynamic UI changes without custom JavaScript.
    - Use **Flowbite** and **Tailwind CSS** for UI components and styling.
    - Support for dark mode and light mode toggling via Tailwind and Flowbite.

---

### **Database Support**

23. **Database Connectivity**:
    - Optionally connect to multiple databases such as:
      - **MongoDB**, **Postgres**, **Amazon DynamoDB**, and **Amazon RDS**.
    - Automatically establish connections based on provided credentials.
    - No limit to the number of connected databases (can be zero, one, or multiple).
    - Database connections should be made available to route handlers.





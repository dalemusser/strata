### **Strata Server Installation Guide**

This guide will walk you through the installation and setup process for **Strata Server**, covering the basic steps for both local development and production deployment.

---

### **Prerequisites**

Before you begin, ensure that you have the following:

- **Golang 1.20 or higher**: You can download the latest version of Go [here](https://golang.org/dl/).
- **Git**: A version control tool to clone the Strata Server repository. You can download Git [here](https://git-scm.com/).
- **AWS Credentials** (if you want to enable CloudFront or CloudWatch logging).

---

### **Step 1: Install Go (golang)**

Download and install Go for your operating system.

[https://go.dev/dl/](https://go.dev/dl/)

---

### **Step 2: Clone the Strata Server Repository**

```bash
git https://github.com/dalemusser/strata.git
cd strata
```

This will download the Strata Server code to your local machine and navigate into the project directory.

---


### **Step 3: Install Dependencies**

Ensure you have all the necessary dependencies by running:

```bash
go mod tidy
```

This will download all the required Go modules.

---

### **Step 3: Configuration**

Strata Server allows you to configure its behavior through **command-line flags**, **environment variables**, and **configuration files** (YAML, JSON, or TOML). The default configuration file is `strata_config.toml`.

#### **Option 1: Using a Configuration File**

1. Create a configuration file called `strata_config.yaml` (or `strata_config.json` or `strata_config.toml`) in the root directory. Here’s an example:

   **strata_config.yaml**
   ```yaml
   host: "localhost"
   port: "8080"
   ssl_env: "test"           # Use "test" for Let's Encrypt staging, "prod" for production
   cert_file: ""
   key_file: ""
   log_file: "server.log"
   static_dir: "./static"
   cloudfront_url: ""        # Add CloudFront URL if using CDN for static files
   shutdown_timeout: 30
   ```

2. The server will automatically use this configuration unless overridden by environment variables or command-line flags.

#### **Option 2: Using Environment Variables**

You can override configuration settings by setting environment variables:

```bash
export STRATA_HOST=localhost
export STRATA_PORT=8080
export STRATA_SSL_ENV=test
export STRATA_STATIC_DIR=./static
```

#### **Option 3: Using Command-Line Flags**

You can also pass settings directly when starting the server:

```bash
go run main.go --host=localhost --port=8080 --ssl_env=test
```

---

### **Step 4: Running Strata Server**

To run the server in **development mode**, which uses HTTP on localhost:

```bash
go run main.go
```

If you want to run it with a custom configuration file, specify the file:

```bash
go run main.go --config=path/to/strata_config.yaml
```

For **production mode** with Let's Encrypt SSL certificates:

1. Ensure you are using a domain name and have DNS correctly set up to point to your server.
2. Set `ssl_env` to `prod` and provide a valid domain in your configuration.

Run the server in production mode:

```bash
go run main.go --ssl_env=prod --host=yourdomain.com
```

### Compiling an Executable Instead of Using 'go run'

Instead of using 'go run', you can compile an executable binary for your operating system by using:

```bash
go build .
```

After the build is complete, an executable called 'strata' will be created. To run the servfer app use the following instead of 'go run':

```bash
./strata
```

Command line parameters can be set the same as shown in the prior examples.

```bash
./strata --host 127.0.0.1 --port 8082
```

Go can compile to a target platform different than the plaform the code is being compiled on.

The following builds for Windows:

```bash
GOOS=windows go build
```

[https://go.dev/doc/install/source#environment](https://go.dev/doc/install/source#environment)

$GOOS and $GOARCH
The name of the target operating system and compilation architecture. These default to the values of $GOHOSTOS and $GOHOSTARCH respectively (described below).

Choices for $GOOS are android, darwin, dragonfly, freebsd, illumos, ios, js, linux, netbsd, openbsd, plan9, solaris, wasip1, and windows.

Choices for $GOARCH are amd64 (64-bit x86, the most mature port), 386 (32-bit x86), arm (32-bit ARM), arm64 (64-bit ARM), ppc64le (PowerPC 64-bit, little-endian), ppc64 (PowerPC 64-bit, big-endian), mips64le (MIPS 64-bit, little-endian), mips64 (MIPS 64-bit, big-endian), mipsle (MIPS 32-bit, little-endian), mips (MIPS 32-bit, big-endian), s390x (IBM System z 64-bit, big-endian), and wasm (WebAssembly 32-bit).


---

### **Step 5: Serving Static Files**

Static files (e.g., images, CSS, JavaScript) are served from the directory specified in the configuration (`static_dir`). By default, this is `./static`.

Place your static files (e.g., `index.html`) in the `./static` directory, and they will be served at the root URL.

For example:
- **File**: `./static/image1.png`
- **URL**: `http://localhost:8080/image1.png`

If you want to use **AWS CloudFront** to serve static files, set the `cloudfront_url` in your configuration:

```yaml
cloudfront_url: "https://your-cloudfront-url"
```

The server will redirect static file requests to CloudFront if this is set.

---

### **Step 6: Enable HTTPS**

By default, Strata Server automatically handles HTTPS with Let's Encrypt.

#### **Using Let's Encrypt (Automatic Certificates)**

In production, set `ssl_env` to `prod` and point your domain to the server:

```bash
go run main.go --ssl_env=prod --host=yourdomain.com
```

Strata Server will automatically generate and manage SSL certificates for your domain.

#### **Using Manual Certificates**

If you want to use your own SSL certificates, specify the `cert_file` and `key_file` in the configuration or environment:

```yaml
cert_file: "path/to/your_cert.pem"
key_file: "path/to/your_key.pem"
```

---

### **Step 7: Logging**

Strata Server logs to a file by default. You can specify the log file path in the configuration file:

```yaml
log_file: "server.log"
```

If you have **AWS CloudWatch** credentials set as environment variables, the logs will also be sent to CloudWatch:

```bash
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export AWS_REGION=your-region
```

---

### **Step 8: Graceful Shutdown**

Strata Server supports graceful shutdown to allow active connections to complete before shutting down. To initiate a graceful shutdown, send a SIGINT or SIGTERM signal to the server (e.g., pressing `Ctrl+C`).

---

### **Step 9: JWT Authentication**

Strata Server uses **JWT** (JSON Web Token) for authentication. Tokens are passed in the `Authorization` header of requests.

To test JWT authentication, log in using one of the supported OAuth providers (e.g., Google, GitHub). The server will issue a JWT which is then used to access protected routes.

---

### **Step 10: OAuth2 Authentication Providers**

Strata Server supports OAuth2 login via multiple providers, such as:
- Google
- GitHub
- Microsoft
- Facebook
- Twitter
- ClassLink

You must set up the required OAuth credentials in environment variables or the configuration file:

```bash
export GOOGLE_CLIENT_ID=your-client-id
export GOOGLE_CLIENT_SECRET=your-client-secret
```

You can add additional providers by configuring them in the environment or configuration.

---

### **Conclusion**

Strata Server is now installed and running! You can adjust the configuration, integrate additional OAuth providers, serve static files, and use CloudFront or local static files. It's a flexible, modern server that meets the needs of developers and production environments alike.

For further assistance, check the documentation and logs for troubleshooting tips, and enjoy using Strata Server!
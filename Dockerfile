FROM golang:1.13-stretch

# Install node and yarn
RUN curl -sL https://deb.nodesource.com/setup_12.x | bash -
RUN apt-get update && apt-get install -y nodejs
RUN npm install -g yarn

# Download golangci-lint and sql.js
WORKDIR /test
RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.24.0
ENV GOOS js
ENV GOARCH wasm
ADD package.json .
RUN yarn install

# Run the tests
ADD . .
CMD go test -exec="./go_sqlite_js_wasm_exec" .

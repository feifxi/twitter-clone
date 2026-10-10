# Build stage: the Maven wrapper pins Maven; dependencies are cached until pom.xml changes.
FROM eclipse-temurin:25-jdk AS builder
WORKDIR /app
COPY .mvn .mvn
COPY mvnw pom.xml ./
RUN ./mvnw -q dependency:go-offline
COPY src src
RUN ./mvnw -q package -DskipTests && \
    java -Djarmode=tools -jar target/twitter-java-api-*.jar extract --layers --launcher --destination /out

# Runtime: JRE only, non-root. Spring Boot's layers put dependencies (rarely changing)
# below the application classes, so rebuilds only replace the small top layers.
FROM eclipse-temurin:25-jre-alpine
WORKDIR /app
RUN addgroup -S app && adduser -S app -G app
COPY --from=builder /out/dependencies/ ./
COPY --from=builder /out/spring-boot-loader/ ./
COPY --from=builder /out/snapshot-dependencies/ ./
COPY --from=builder /out/application/ ./
USER app
EXPOSE 8080
ENTRYPOINT ["java", "-XX:MaxRAMPercentage=75", "org.springframework.boot.loader.launch.JarLauncher"]

plugins {
    id("com.android.application")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

android {
    namespace = "com.mandarinkb.cycling_navigator"
    compileSdk = flutter.compileSdkVersion
    ndkVersion = flutter.ndkVersion

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    defaultConfig {
        // TODO: Specify your own unique Application ID (https://developer.android.com/studio/build/application-id.html).
        applicationId = "com.mandarinkb.cycling_navigator"
        // You can update the following values to match your application needs.
        // For more information, see: https://flutter.dev/to/review-gradle-config.
        // valhalla-mobile requires API 26+ (Android 8.0, 2017) — a fine
        // floor for this app regardless.
        minSdk = 26
        targetSdk = flutter.targetSdkVersion
        // Uses the version code from pubspec.yaml. When using split APKs, 1000 * ABI_VERSION
        // is added automatically by Flutter. (https://developer.android.com/studio/build/configure-apk-splits#configure-APK-versions)
        // You can force using the value of versionCode by specifying the `-P force-version-code-ignoring-abi=true`
        // flag during build.
        versionCode = flutter.versionCode
        versionName = flutter.versionName
    }

    buildTypes {
        release {
            // TODO: Add your own signing config for the release build.
            // Signing with the debug keys for now, so `flutter run --release` works.
            signingConfig = signingConfigs.getByName("debug")
        }
    }
}

kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
    }
}

flutter {
    source = "../.."
}

dependencies {
    // Phase 6a spike: does a Valhalla mobile build link into a real Android
    // app at all? io.github.rallista publishes a prebuilt AAR to Maven
    // Central, so this doesn't require cross-compiling Valhalla ourselves —
    // see docs/valhalla-mobile at github.com/Rallista/valhalla-mobile.
    implementation("io.github.rallista:valhalla-mobile:0.6.1")
    implementation("io.github.rallista:valhalla-models:0.5.2")
    implementation("io.github.rallista:valhalla-models-config:0.5.2")
    implementation("com.stadiamaps:osrm-openapi:0.0.10")
}

package com.emulator;

import java.io.File;
import java.net.URL;
import java.net.URLClassLoader;
import javax.microedition.midlet.MIDlet;

public class Main {
    private static MIDlet currentMidlet;

    public static ClassLoader j2meClassLoader;

    public static void startApp(String jarPath) {
        try {
            System.out.println("[Main] Loading J2ME JAR: " + jarPath);
            File file = new File(jarPath);
            if (!file.exists()) {
                System.err.println("[Main] ERROR: JAR file does not exist: " + jarPath);
                return;
            }

            URLClassLoader classLoader = new URLClassLoader(
                new URL[]{ file.toURI().toURL() },
                Main.class.getClassLoader()
            ) {
                protected Class<?> loadClass(String name, boolean resolve) throws ClassNotFoundException {
                    Class<?> c = findLoadedClass(name);
                    if (c == null) {
                        if (name.startsWith("java.") || name.startsWith("javax.microedition.") || name.startsWith("com.emulator.") || name.equals("aq") || name.equals("x2.aq") || name.startsWith("CLib.mImage") || name.startsWith("CLib.TemGraphics")) {
                            return super.loadClass(name, resolve);
                        }
                        try {
                            c = findClass(name);
                        } catch (ClassNotFoundException e) {
                            c = super.loadClass(name, resolve);
                        }
                    }
                    if (resolve) {
                        resolveClass(c);
                    }
                    return c;
                }
            };

            j2meClassLoader = classLoader;
            Thread.currentThread().setContextClassLoader(classLoader);

            String midletClassName = "com.silverknight.TemMidlet";
            try {
                java.util.jar.JarFile jarFile = new java.util.jar.JarFile(file);
                java.util.jar.Manifest manifest = jarFile.getManifest();
                if (manifest != null) {
                    String midletAttr = manifest.getMainAttributes().getValue("MIDlet-1");
                    if (midletAttr != null) {
                        String[] parts = midletAttr.split(",");
                        if (parts.length >= 3) {
                            midletClassName = parts[2].trim();
                        }
                    }
                }
                jarFile.close();
            } catch (Exception e) {
                System.out.println("[Main] Notice: Manifest read fallback to " + midletClassName + ": " + e.getMessage());
            }

            Class<?> midletClass = classLoader.loadClass(midletClassName);
            System.out.println("[Main] Loaded MIDlet class: " + midletClass.getName());

            MIDlet midlet = (MIDlet) midletClass.getDeclaredConstructor().newInstance();
            currentMidlet = midlet;

            java.lang.reflect.Method startAppMethod = midletClass.getDeclaredMethod("startApp");
            startAppMethod.setAccessible(true);
            System.out.println("[Main] Invoking startApp()...");
            startAppMethod.invoke(midlet);
            System.out.println("[Main] MIDlet startApp() executed successfully.");

            // Background Garbage Collector to keep native VRAM memory low
            new Thread(new Runnable() {
                public void run() {
                    while (true) {
                        try {
                            Thread.sleep(5000); // 5 seconds
                            System.gc(); // Force Java to collect unused Image objects
                        } catch (InterruptedException e) {}
                    }
                }
            }).start();
        } catch (Exception e) {
            System.err.println("[Main] Exception starting MIDlet:");
            e.printStackTrace();
        }
    }

    public static void main(String[] args) {
        String path = (args != null && args.length > 0) ? args[0] : "/home/neuro/Downloads/x2.jar";
        if (!new File(path).exists()) {
            path = "/home/neuro/Downloads/KnightOnline_402.jar";
        }
        startApp(path);
    }
}

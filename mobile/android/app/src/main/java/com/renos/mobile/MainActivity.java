package com.renos.mobile;

import android.annotation.SuppressLint;
import android.app.Activity;
import android.content.SharedPreferences;
import android.graphics.Color;
import android.graphics.Insets;
import android.os.Build;
import android.os.Bundle;
import android.view.Gravity;
import android.view.ViewGroup;
import android.view.Window;
import android.view.WindowInsets;
import android.view.WindowInsetsController;
import android.view.inputmethod.EditorInfo;
import android.view.inputmethod.InputMethodManager;
import android.text.InputType;
import android.webkit.WebChromeClient;
import android.webkit.CookieManager;
import android.webkit.WebSettings;
import android.webkit.WebStorage;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.Button;
import android.widget.EditText;
import android.widget.LinearLayout;

public class MainActivity extends Activity {
    private static final String PREFS = "renos";
    private static final String KEY_URL = "hub_url";
    private static final String DEFAULT_URL = "http://192.168.0.117:18081";
    private static final String OLD_DEFAULT_URL = "https://renos.local";
    private WebView webView;
    private EditText urlInput;

    @Override
    @SuppressLint("SetJavaScriptEnabled")
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        SharedPreferences prefs = getSharedPreferences(PREFS, MODE_PRIVATE);
        String savedUrl = prefs.getString(KEY_URL, DEFAULT_URL);
        boolean migratedLegacyUrl = false;
        if (OLD_DEFAULT_URL.equals(savedUrl)) {
            savedUrl = DEFAULT_URL;
            prefs.edit().putString(KEY_URL, savedUrl).apply();
            clearLegacyWebViewState();
            migratedLegacyUrl = true;
        }

        LinearLayout root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        root.setBackgroundColor(Color.rgb(15, 17, 23));
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            root.setOnApplyWindowInsetsListener((view, insets) -> {
                int bars = WindowInsets.Type.systemBars() | WindowInsets.Type.displayCutout();
                int handled = bars | WindowInsets.Type.ime();
                Insets padding = insets.getInsets(handled);
                view.setPadding(padding.left, padding.top, padding.right, padding.bottom);
                // The native container handles these areas, so WebView must not add them again.
                return new WindowInsets.Builder(insets)
                        .setInsets(handled, Insets.NONE)
                        .setInsetsIgnoringVisibility(bars, Insets.NONE)
                        .setDisplayCutout(null)
                        .build();
            });
        } else {
            root.setFitsSystemWindows(true);
        }

        LinearLayout bar = new LinearLayout(this);
        bar.setOrientation(LinearLayout.HORIZONTAL);
        bar.setGravity(Gravity.CENTER_VERTICAL);
        int pad = dp(8);
        bar.setPadding(pad, pad, pad, pad);

        urlInput = new EditText(this);
        urlInput.setSingleLine(true);
        urlInput.setTextColor(Color.WHITE);
        urlInput.setHintTextColor(Color.LTGRAY);
        urlInput.setTextSize(16);
        urlInput.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_URI);
        urlInput.setImeOptions(EditorInfo.IME_ACTION_GO);
        urlInput.setOnEditorActionListener((view, action, event) -> {
            if (action == EditorInfo.IME_ACTION_GO) {
                loadFromInput();
                return true;
            }
            return false;
        });
        urlInput.setText(savedUrl);
        urlInput.setHint(DEFAULT_URL);
        urlInput.setLayoutParams(new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1));

        Button connectButton = new Button(this);
        connectButton.setText("Open");
        connectButton.setOnClickListener(v -> loadFromInput());

        bar.addView(urlInput);
        bar.addView(connectButton);

        webView = new WebView(this);
        webView.setLayoutParams(new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 0, 1));
        webView.setWebViewClient(new WebViewClient());
        webView.setWebChromeClient(new WebChromeClient());

        WebSettings settings = webView.getSettings();
        settings.setJavaScriptEnabled(true);
        settings.setDomStorageEnabled(true);
        settings.setDatabaseEnabled(true);
        settings.setMediaPlaybackRequiresUserGesture(false);
        // Respect the page viewport: a forced scale retains login focus panning after navigation.
        settings.setLoadWithOverviewMode(true);
        settings.setUseWideViewPort(true);
        settings.setBuiltInZoomControls(false);
        settings.setDisplayZoomControls(false);
        settings.setTextZoom(100);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
            settings.setMixedContentMode(WebSettings.MIXED_CONTENT_COMPATIBILITY_MODE);
        }
        if (migratedLegacyUrl) {
            webView.clearCache(true);
            webView.clearHistory();
        }

        root.addView(bar);
        root.addView(webView);
        setContentView(root);
        configureSystemBars();
        root.requestApplyInsets();

        webView.loadUrl(normalizeUrl(savedUrl));
    }

    private void loadFromInput() {
        String url = normalizeUrl(urlInput.getText().toString());
        getSharedPreferences(PREFS, MODE_PRIVATE).edit().putString(KEY_URL, url).apply();
        InputMethodManager keyboard = (InputMethodManager) getSystemService(INPUT_METHOD_SERVICE);
        keyboard.hideSoftInputFromWindow(urlInput.getWindowToken(), 0);
        urlInput.clearFocus();
        webView.requestFocus();
        webView.loadUrl(url);
    }

    private String normalizeUrl(String raw) {
        String value = raw == null ? "" : raw.trim();
        if (value.isEmpty()) {
            return DEFAULT_URL;
        }
        if (!value.startsWith("http://") && !value.startsWith("https://")) {
            return "http://" + value;
        }
        return value;
    }

    private int dp(int value) {
        return Math.round(value * getResources().getDisplayMetrics().density);
    }

    private void configureSystemBars() {
        Window window = getWindow();
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            window.setDecorFitsSystemWindows(false);
            WindowInsetsController controller = window.getInsetsController();
            if (controller != null) {
                controller.setSystemBarsAppearance(0,
                        WindowInsetsController.APPEARANCE_LIGHT_STATUS_BARS
                                | WindowInsetsController.APPEARANCE_LIGHT_NAVIGATION_BARS);
            }
        }
        window.setStatusBarColor(Color.rgb(15, 17, 23));
        window.setNavigationBarColor(Color.rgb(15, 17, 23));
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            android.view.WindowManager.LayoutParams attrs = window.getAttributes();
            attrs.layoutInDisplayCutoutMode =
                    android.view.WindowManager.LayoutParams.LAYOUT_IN_DISPLAY_CUTOUT_MODE_SHORT_EDGES;
            window.setAttributes(attrs);
        }
    }

    private void clearLegacyWebViewState() {
        CookieManager.getInstance().removeAllCookies(null);
        CookieManager.getInstance().flush();
        WebStorage.getInstance().deleteAllData();
        WebView.clearClientCertPreferences(null);
    }

    @Override
    public void onBackPressed() {
        if (webView != null && webView.canGoBack()) {
            webView.goBack();
            return;
        }
        super.onBackPressed();
    }
}

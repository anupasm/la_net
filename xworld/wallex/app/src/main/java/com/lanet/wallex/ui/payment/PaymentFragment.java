package com.lanet.wallex.ui.payment;

import android.content.Intent;
import android.os.Bundle;
import android.util.Log;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Button;
import android.widget.ProgressBar;
import android.widget.Toast;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.fragment.app.Fragment;

import com.google.zxing.integration.android.IntentIntegrator;
import com.google.zxing.integration.android.IntentResult;
import com.lanet.wallex.R;
import com.lanet.wallex.databinding.FragmentPaymentBinding;

import org.json.JSONException;
import org.json.JSONObject;

import java.net.URI;
import java.net.URISyntaxException;

import org.java_websocket.client.WebSocketClient;
import org.java_websocket.handshake.ServerHandshake;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class PaymentFragment extends Fragment {

    private static final Logger log = LoggerFactory.getLogger(PaymentFragment.class);
    private FragmentPaymentBinding binding;
    private WebSocketClient wsclient;
    public static final String MOBILE_CW_ENDPOINT = "ws://192.168.1.6:9052/pay";
    private PaymentViewModel model;
    private Button btnPay;
    private ProgressBar pb;

    public View onCreateView(@NonNull LayoutInflater inflater,
                             ViewGroup container, Bundle savedInstanceState) {

        binding = FragmentPaymentBinding.inflate(inflater, container, false);
        View root = binding.getRoot();
        model = new PaymentViewModel();

        btnPay = (Button) root.findViewById(R.id.btn_pay);
        pb = (ProgressBar) root.findViewById(R.id.progress);
        pb.setVisibility(View.INVISIBLE);
        btnPay.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                String merchant = model.getMerchant();
                String amount =model.getAmount();
                String sid=model.getSid();
                String secret=model.getSecret();
                pb.setVisibility(View.VISIBLE);
                createWebSocketClient(merchant,amount,sid,secret);

            }
        });

        IntentIntegrator integrator = IntentIntegrator.forSupportFragment(PaymentFragment.this);
        integrator.setOrientationLocked(false);
        integrator.setPrompt("Scan QR code");
        integrator.setBeepEnabled(false);
        integrator.setDesiredBarcodeFormats(IntentIntegrator.QR_CODE);
        integrator.initiateScan();
        return root;
    }

    @Override
    public void onDestroyView() {
        super.onDestroyView();
        binding = null;
    }

    @Override
    public void onActivityResult(int requestCode, int resultCode, @Nullable Intent data) {
        IntentResult result = IntentIntegrator.parseActivityResult(requestCode, resultCode, data);
        if(result != null) {
            if(result.getContents() == null) {
                Toast.makeText(getContext(), "Cancelled", Toast.LENGTH_LONG).show();
            } else {
                JSONObject obj = null;
                try {
                    obj = new JSONObject(result.getContents());
                    String merchant = obj.getString("Merchant");
                    binding.textMerchant.setText(merchant);
                    String amount = obj.getString("Amount");
                    binding.textAmount.setText(amount);
                    String secret = obj.getString("Secret");
                    String sid = obj.getString("SID");
                    model.setAmount(amount);
                    model.setMerchant(merchant);
                    model.setSecret(secret);
                    model.setSid(sid);
                } catch (JSONException e) {
                    throw new RuntimeException(e);
                }
            }
        }
    }


    private void createWebSocketClient(String merchant, String amount,String sid,String secret) {
        URI uri;
        try {
            uri = new URI(MOBILE_CW_ENDPOINT);
        }
        catch (URISyntaxException e) {
            Log.e("Websocket",e.getMessage());
            return;
        }

        wsclient = new WebSocketClient(uri) {
            @Override
            public void onOpen(ServerHandshake handshakedata) {
                Log.i("WebSocket", "Session is starting");

                JSONObject json = new JSONObject();
                try {
                    json.put("Merchant", merchant);
                    json.put("Amount", amount);
                    json.put("SID", sid);
                    json.put("Secret", secret);
                    wsclient.send(json.toString());
                } catch (JSONException e) {
                    throw new RuntimeException(e);
                }
            }

            @Override
            public void onMessage(String message) {
                getActivity().runOnUiThread(new Runnable() {
                    @Override
                    public void run() {
                        try{
                            Toast.makeText(getContext(),message,Toast.LENGTH_LONG).show();
                            pb.setVisibility(View.INVISIBLE);
                        } catch (Exception e){
                            e.printStackTrace();
                        }
                    }
                });
            }

            @Override
            public void onClose(int code, String reason, boolean remote) {
                Log.i("WebSocket", reason);
            }

            @Override
            public void onError(Exception ex) {
                Log.e("WebSocket", ex.toString());
            }
        };

        wsclient.connect();
    }


}
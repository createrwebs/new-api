package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/thanhpk/randstr"
)

const (
	PromptPayTargetPhone = "0639304353"
	// Quota units credited per 1 THB (standard 1 USD = 35 THB = 500,000 units => ~14,285 units; rounded to 15,000 units/THB)
	PromptPayQuotaPerTHB = 15000
)

type PromptPayPayRequest struct {
	Amount float64 `json:"amount"` // Amount in Thai Baht (THB)
}

type PromptPaySlipRequest struct {
	TradeNo  string `json:"trade_no"`
	SlipNote string `json:"slip_note"`
	SlipUrl  string `json:"slip_url"`
}

// crc16CCITT calculates the EMVCo standard CRC16 checksum using polynomial 0x1021 with init 0xFFFF
func crc16CCITT(data string) string {
	var crc uint16 = 0xFFFF
	for i := 0; i < len(data); i++ {
		crc ^= uint16(data[i]) << 8
		for j := 0; j < 8; j++ {
			if (crc & 0x8000) != 0 {
				crc = ((crc << 1) ^ 0x1021) & 0xFFFF
			} else {
				crc = (crc << 1) & 0xFFFF
			}
		}
	}
	return fmt.Sprintf("%04X", crc)
}

// GenPromptPayPayload generates an EMVCo compliant Thai QR Payment PromptPay payload string
func GenPromptPayPayload(phone string, amount float64) string {
	digits := ""
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits += string(r)
		}
	}

	var target string
	if len(digits) == 10 && digits[0] == '0' {
		target = "0066" + digits[1:]
	} else if len(digits) == 13 {
		target = digits
	} else {
		target = "0066" + digits
	}

	subtag := fmt.Sprintf("01%02d%s", len(target), target)
	aid := "0016A000000677010111"
	merchantData := aid + subtag
	tag29 := fmt.Sprintf("29%02d%s", len(merchantData), merchantData)

	var raw string
	if amount > 0 {
		amtStr := fmt.Sprintf("%.2f", amount)
		raw = fmt.Sprintf("000201010212%s530376454%02d%s5802TH6304", tag29, len(amtStr), amtStr)
	} else {
		raw = fmt.Sprintf("000201010211%s53037645802TH6304", tag29)
	}

	return raw + crc16CCITT(raw)
}

// RequestPromptPay handles PromptPay payment initialization for Thai users
func RequestPromptPay(c *gin.Context) {
	var req PromptPayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "ข้อมูลคำขอไม่ถูกต้อง"})
		return
	}

	if req.Amount < 1 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "ยอดชำระขั้นต่ำ 1 บาท"})
		return
	}
	if req.Amount > 100000 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "ยอดชำระสูงสุดไม่เกิน 100,000 บาทต่อรายการ"})
		return
	}

	id := c.GetInt("id")
	user, err := model.GetUserById(id, false)
	if err != nil || user == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "ไม่พบข้อมูลผู้ใช้"})
		return
	}

	topUpGroupRatio := common.GetTopupGroupRatio(user.Group)
	if topUpGroupRatio == 0 {
		topUpGroupRatio = 1
	}

	quotaToCredit := int64(req.Amount * float64(PromptPayQuotaPerTHB) * topUpGroupRatio)

	if rejectInvalidCreditedQuota(c, id, decimal.NewFromInt(quotaToCredit)) {
		return
	}

	tradeNo := fmt.Sprintf("pp_%d_%d_%s", user.Id, time.Now().Unix(), randstr.String(4))
	qrPayload := GenPromptPayPayload(PromptPayTargetPhone, req.Amount)

	topUp := &model.TopUp{
		UserId:          id,
		Amount:          quotaToCredit,
		Money:           req.Amount,
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodPromptPay,
		PaymentProvider: model.PaymentProviderPromptPay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}

	err = topUp.Insert()
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("PromptPay 创建充值订单失败 user_id=%d trade_no=%s amount=%.2f error=%q", id, tradeNo, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "สร้างรายการชำระเงินไม่สำเร็จ"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("PromptPay 订单创建成功 user_id=%d trade_no=%s amount=%.2f quota=%d", id, tradeNo, req.Amount, quotaToCredit))

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"trade_no":        tradeNo,
			"phone":           PromptPayTargetPhone,
			"formatted_phone": "063-930-4353",
			"amount":          req.Amount,
			"currency":        "THB",
			"qr_payload":      qrPayload,
			"quota":           quotaToCredit,
			"instruction":     "โอนเงินผ่านพร้อมเพย์เบอร์ 063-930-4353 หรือสแกน QR Code แล้วแจ้งรหัสอ้างอิง",
		},
	})
}

// SubmitPromptPaySlip handles submission of transaction slip / note for PromptPay verification
func SubmitPromptPaySlip(c *gin.Context) {
	var req PromptPaySlipRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "กรุณาระบุรหัสอ้างอิงคำสั่งซื้อ"})
		return
	}

	id := c.GetInt("id")
	tradeNo := strings.TrimSpace(req.TradeNo)

	var topUp model.TopUp
	err := model.DB.Where("trade_no = ? AND user_id = ?", tradeNo, id).First(&topUp).Error
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "ไม่พบรายการชำระเงินที่ระบุ"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("PromptPay 用户提交转账凭证 user_id=%d trade_no=%s note=%s url=%s", id, tradeNo, req.SlipNote, req.SlipUrl))

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"trade_no": tradeNo,
			"status":   topUp.Status,
			"message":  "บันทึกข้อมูลการแจ้งโอนเรียบร้อย ระบบจะตรวจสอบและเพิ่มเครดิตเข้ากระเป๋าของคุณ",
		},
	})
}

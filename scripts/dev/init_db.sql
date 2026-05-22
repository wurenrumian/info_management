BEGIN;

DO $$
DECLARE
  mode text := COALESCE(NULLIF(current_setting('seed_mode', true), ''), 'all');
BEGIN
  IF mode IN ('all', 'notification') THEN
    INSERT INTO "notification_templates"
    ("code", "wechat_template_id", "name", "created_at", "updated_at")
    VALUES
      ('time_reminder', 'zk-fz3D2ggWyttyPFQm3UReRiHkxu_0Hn0cOCx3nijk', '活动执行时间提醒', '2026-04-06 00:00:00+00', '2026-04-06 00:00:00+00'),
      ('approval_result', 'M626iRqFxkQwDX81ASfDsZzCHUjs0MQAjPpKWDMUAig', '审批结果通知', '2026-04-06 00:00:00+00', '2026-04-06 00:00:00+00'),
      ('login_sucessful', 'tDO0vkDRDxFHbgIYcuS8U9BY_cAMylqyJlj8Q0HeQpw', '登陆成功通知', '2026-04-06 00:00:00+00', '2026-04-06 00:00:00+00'),
      ('announcement', 'GBh26deOiePPxI4-0N_0URV7-XGJDhV8EViUjlyz-_Q', '平台公告通知', '2026-04-06 00:00:00+00', '2026-04-06 00:00:00+00')
    ON CONFLICT ("code") DO NOTHING;
  END IF;

  IF mode IN ('all', 'certificate') THEN
    INSERT INTO "certificate_templates"
    ("code", "name", "approval_type", "document_stage", "status", "renderer", "template_path", "template_version", "field_mapping", "disclaimer", "created_at", "updated_at")
    VALUES
      ('leave_application_pdf', '请假申请材料 PDF', 'leave', 'application', 'active', 'typst', 'templates/certificates/leave_application.typ', 'v1', '{}'::jsonb, '', '2026-04-06 00:00:00+00', '2026-04-06 00:00:00+00'),
      ('leave_approval_certificate', '请假审批结果凭证', 'leave', 'approval_certificate', 'active', 'typst', 'templates/certificates/leave_approval_certificate.typ', 'v1', '{}'::jsonb, '本文件由学院信息管理系统自动生成，仅用于学院内部审批留痕与流转，不等同于学校正式公章文件。', '2026-04-06 00:00:00+00', '2026-04-06 00:00:00+00'),
      ('budget_application_pdf', '预算申请材料 PDF', 'budget', 'application', 'active', 'typst', 'templates/certificates/budget_application.typ', 'v1', '{}'::jsonb, '', '2026-04-06 00:00:00+00', '2026-04-06 00:00:00+00'),
      ('budget_approval_certificate', '预算审批结果凭证', 'budget', 'approval_certificate', 'active', 'typst', 'templates/certificates/budget_approval_certificate.typ', 'v1', '{}'::jsonb, '本文件由学院信息管理系统自动生成，仅用于学院内部审批留痕与流转，不等同于学校正式公章文件。', '2026-04-06 00:00:00+00', '2026-04-06 00:00:00+00')
    ON CONFLICT ("code") DO NOTHING;
  END IF;
END $$;

COMMIT;

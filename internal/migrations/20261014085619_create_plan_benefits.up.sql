-- Nilai benefit per plan (spec §2.4). Composite FK menjamin plan & benefit
-- berasal dari solution yang sama.
CREATE TABLE plan_benefits (
    plan_id     UUID  NOT NULL,
    benefit_id  UUID  NOT NULL,
    solution_id UUID  NOT NULL,
    value       JSONB NOT NULL,

    CONSTRAINT pk_plan_benefits PRIMARY KEY (plan_id, benefit_id),
    CONSTRAINT fk_plan_benefits_plan FOREIGN KEY (plan_id, solution_id) REFERENCES plans (id, solution_id),
    CONSTRAINT fk_plan_benefits_benefit FOREIGN KEY (benefit_id, solution_id) REFERENCES benefits (id, solution_id)
);

CREATE INDEX idx_plan_benefits_benefit_id ON plan_benefits (benefit_id);

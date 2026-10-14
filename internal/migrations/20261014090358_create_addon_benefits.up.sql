-- Nilai benefit per add-on (spec §2.4). Composite FK menjamin add-on & benefit
-- berasal dari solution yang sama.
CREATE TABLE addon_benefits (
    addon_id    UUID  NOT NULL,
    benefit_id  UUID  NOT NULL,
    solution_id UUID  NOT NULL,
    value       JSONB NOT NULL,

    CONSTRAINT pk_addon_benefits PRIMARY KEY (addon_id, benefit_id),
    CONSTRAINT fk_addon_benefits_addon FOREIGN KEY (addon_id, solution_id) REFERENCES addons (id, solution_id),
    CONSTRAINT fk_addon_benefits_benefit FOREIGN KEY (benefit_id, solution_id) REFERENCES benefits (id, solution_id)
);

CREATE INDEX idx_addon_benefits_benefit_id ON addon_benefits (benefit_id);
